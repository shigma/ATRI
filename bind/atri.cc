#include <node.h>
#include <uv.h>
#include <string.h>
#include <fstream>
#include <ios>
#include <functional>
#include <iostream>
#include <atomic>
#include <stdexcept>
#include "../src/main.h"

#define ENSURE_UV(x) assert(x == 0);

namespace ATRI {
	using v8::Array;
	using v8::Exception;
	using v8::External;
	using v8::Context;
	using v8::Function;
	using v8::FunctionCallbackInfo;
	using v8::FunctionTemplate;
	using v8::Isolate;
	using v8::Local;
	using v8::MaybeLocal;
	using v8::Number;
	using v8::Integer;
	using v8::Object;
	using v8::ObjectTemplate;
	using v8::PropertyCallbackInfo;
	using v8::String;
	using v8::Value;

	struct CharUtil
	{
		String::Utf8Value* str;
		CharUtil(Isolate* isolate, Local<Value> value): str(new String::Utf8Value(isolate, value)) {}

		operator char* () { return **str; }

		~CharUtil() { delete str; }
	};

	enum class Pattern
	{
		PLAIN,
		CONSTRUCTOR,
		INSTANCE_SYNC,
		INSTANCE_ASYNC,
		INSTANCE_LISTENER,
	};

	inline void TypeError(Isolate* isolate, const char* message) {
		isolate->ThrowException(Exception::TypeError(String::NewFromUtf8(isolate, message).ToLocalChecked()));
	}

	template<void* F, Pattern P, size_t N, typename... A, size_t... S>
	inline void V8CallbackNumbered(const FunctionCallbackInfo<Value>& args, std::index_sequence<S...>) {
		Isolate* isolate = args.GetIsolate();
		constexpr size_t REAL_N = (P == Pattern::INSTANCE_ASYNC || P == Pattern::INSTANCE_LISTENER) ? N + 1 : N;
		if (args.Length() < REAL_N) {
			return TypeError(isolate, "missing arguments");
		}

		if constexpr (P == Pattern::CONSTRUCTOR) {
			if (!args.IsConstructCall()) {
				return TypeError(isolate, "Constructor requires 'new'");
			}

			void* bot;
			try {
				bot = reinterpret_cast<void*>(F(std::forward<A>(Convert<A>(isolate, args[S]))...));
			}
			catch (std::invalid_argument& ex) {
				return TypeError(isolate, ex.what());
			}

			if (bot == nullptr) {
				return TypeError(isolate, "Constructor requires 'new'");
			}

			const auto This = args.This();
			This->SetAlignedPointerInInternalField(0, bot);
		}
		else if constexpr (P == Pattern::INSTANCE_SYNC || P == Pattern::INSTANCE_ASYNC || P == Pattern::INSTANCE_LISTENER) {
			const auto This = args.This();
			if (This->InternalFieldCount() != 1) {
				return TypeError(isolate, "Wrong context");
			}
			void* bot = This->GetAlignedPointerFromInternalField(0);
			if (bot == nullptr) {
				return TypeError(isolate, "Wrong context");
			}
			Local<Context> ctx = isolate->GetCurrentContext();

			if constexpr (P == Pattern::INSTANCE_SYNC) {
				char* result;
				try {
					result = F(bot, std::forward<A>(Convert<A>(isolate, args[S]))...);
				}
				catch (std::invalid_argument& ex) {
					return TypeError(isolate, ex.what());
				}
				args.GetReturnValue().Set(ToJSON(isolate, ctx, result));
				// 这里一 free 就崩溃
				// “就一点内存，泄露就泄露了”——jjyyxx
				// “或者 c 处理完以后 notify go”——西格玛
				// “妙哉”——jjyyxx
				// 至于具体是否有效果，之后可以用一些大字符串检验
				GoFree(result);
			}
			else {
				if constexpr (P == Pattern::INSTANCE_ASYNC) {
					AsyncByteWork* work = new AsyncByteWork(isolate, Convert<Local<Function>>(isolate, args[N]));
					work->Invoke<F>(bot, std::forward<A>(Convert<A>(isolate, args[S]))...);
				}
				else if constexpr (P == Pattern::INSTANCE_LISTENER) {
					ListenerByteWork* work = new ListenerByteWork(isolate, Convert<Local<Function>>(isolate, args[N]));
					work->Invoke<F>(bot, std::forward<A>(Convert<A>(isolate, args[S]))...);
				}
				else {
					// NEVER
					static_assert(false, "Unimplemented");
				}

				args.GetReturnValue().Set(Undefined(isolate));
			}
		}
		else {
			static_assert(false, "Unimplemented");
		}
	}

	template<void* F, Pattern P, typename... A>
	void V8Callback(const FunctionCallbackInfo<Value>& args) {
		constexpr size_t N = sizeof...(A);
		constexpr auto S = std::make_index_sequence<N>{};
		V8CallbackNumbered<F, P, N, A...>(args, S);
	}

	template<void* F, Pattern P, typename... A>
	inline void AddMethod(Isolate* isolate, Local<ObjectTemplate> tpl, const char* name) {
		tpl->Set(isolate, name, FunctionTemplate::New(isolate, V8Callback<F, P, A...>));
	}

	template<typename T>
	T Convert(Isolate* isolate, Local<Value>&& value) {
		if constexpr (std::is_same_v<T, CharUtil>) {
			if (!value->IsString()) throw std::invalid_argument("expect string");
			return { isolate, value };
		}
		else if constexpr (std::is_same_v<T, bool>) {
			if (!value->IsBoolean()) throw std::invalid_argument("expect boolean");
			return value->BooleanValue(isolate);
		}
		else if constexpr (std::is_floating_point_v<T>) {
			if (!value->IsNumber()) throw std::invalid_argument("expect number");
			return value->NumberValue(isolate->GetCurrentContext()).FromJust();
		}
		else if constexpr (std::is_integral_v<T>) {
			if (!value->IsNumber()) throw std::invalid_argument("expect number");
			return value->IntegerValue(isolate->GetCurrentContext()).FromJust();
		}
		else if constexpr (std::is_same_v<T, Local<Function>>) {
			if (!value->IsFunction()) throw std::invalid_argument("expect function");
			return Local<Function>::Cast(value);
		}
	}

	Local<Value> ToJSON(Isolate* isolate, Local<Context> context, char* string) {
		return v8::JSON::Parse(context, String::NewFromUtf8(isolate, string).ToLocalChecked()).ToLocalChecked();
	}

	struct AsyncByteWork {
		uv_async_t request{};
		v8::Persistent<Function> callback;
		AsyncByteWork(Isolate* isolate, Local<Function> callback) : callback(isolate, callback) {
			request.data = this;
			request.close_cb = close_callback_func;
		}

		~AsyncByteWork() {
			delete result;
			delete error;
		}

		template<void* F, typename... Ts>
		void Invoke(Ts... args) {
			ENSURE_UV(uv_async_init(uv_default_loop(), &this->request, this->node_callback_func));
			F(args..., go_callback_func, reinterpret_cast<uintptr_t>(this));
		}

		void Dispose() {
			this->callback.Reset();
			uv_close((uv_handle_t*)&request, NULL);
		}

		char* error = nullptr;
		char* result = nullptr;
		size_t length = -1;
		std::atomic_bool called = false;
		static void go_callback_func(uintptr_t ctx, void* result, void* error, size_t length) {
			AsyncByteWork* work = reinterpret_cast<AsyncByteWork*>(ctx);

			bool expect = false;
			work->called.compare_exchange_strong(expect, true);
			assert(!expect);

			assert((error == nullptr) ^ (result == nullptr)); // one of them
			if (result) {
				memcpy(work->result = new char[length + 1], result, length);
				work->result[length] = '\0';
			}
			else {
				memcpy(work->error = new char[length + 1], error, length);
				work->error[length] = '\0';
			}
			work->length = length;
			ENSURE_UV(uv_async_send(&work->request));
		}

		static void node_callback_func(uv_async_t* request) {
			AsyncByteWork* work = static_cast<AsyncByteWork*>(request->data);

			Isolate* isolate = Isolate::GetCurrent();
			v8::HandleScope handleScope(isolate);

			Local<Context> ctx = isolate->GetCurrentContext();

			Local<Value> argv[2]{
				work->error == nullptr ? static_cast<Local<Value>>(v8::Null(isolate)) : ToJSON(isolate, ctx, work->error),
				work->result == nullptr ? static_cast<Local<Value>>(v8::Null(isolate)) : ToJSON(isolate, ctx, work->result)
			};

			Local<Function>::New(isolate, work->callback)->Call(ctx, ctx->Global(), 2, argv);
			work->Dispose();
		}

		static void close_callback_func(uv_handle_t* request) {
			AsyncByteWork* work = static_cast<AsyncByteWork*>(request->data);
			delete work;
		}

		// For test purpose
		uint64_t now = uv_hrtime();
		void update_and_print(int tag) {
			uint64_t next = uv_hrtime();
			uint64_t duration = next - now;
			std::cout << tag << ":" << duration << std::endl;
			now = uv_hrtime();
		}
	};

	struct ListenerByteWork {
		uv_async_t request{};
		uv_mutex_t mutex{};
		v8::Persistent<Function> callback;
		ListenerByteWork(Isolate* isolate, Local<Function> callback): callback(isolate, callback) {
			request.data = this;
			request.close_cb = close_callback_func;
		}

		~ListenerByteWork() {
			delete buffer;
		}

		template<void* F, typename... Ts>
		void Invoke(Ts... args) {
			this->replace_buffer();
			ENSURE_UV(uv_async_init(uv_default_loop(), &this->request, this->node_callback_func));
			ENSURE_UV(uv_mutex_init(&this->mutex));
			F(args..., go_callback_func, reinterpret_cast<uintptr_t>(this));
		}

		void Dispose() {
			// TODO: much more tricky than Async
			this->callback.Reset();
			uv_close((uv_handle_t*)&request, NULL);
		}

		std::vector<char*>* buffer;
		std::vector<char*>* replace_buffer() {
			auto old = buffer;
			buffer = new std::vector<char*>(); // give a suitable initial size?
			return old;
		}

		static void go_callback_func(uintptr_t ctx, void* result, void* error, size_t length) {
			ListenerByteWork* work = reinterpret_cast<ListenerByteWork*>(ctx);
			assert(error == nullptr);
			char* message = new char[length + 1];
			memcpy(message, result, length);
			message[length] = '\0';
			
			uv_mutex_lock(&work->mutex);
			work->buffer->push_back(message);
			uv_mutex_unlock(&work->mutex);

			ENSURE_UV(uv_async_send(&work->request));
		}

		static void node_callback_func(uv_async_t* request) {
			ListenerByteWork* work = static_cast<ListenerByteWork*>(request->data);

			uv_mutex_lock(&work->mutex);
			const auto result = work->replace_buffer();
			uv_mutex_unlock(&work->mutex);

			Isolate* isolate = Isolate::GetCurrent();
			v8::HandleScope handleScope(isolate);

			Local<Context> ctx = isolate->GetCurrentContext();

			const size_t length = result->size();
			Local<Value>* valueArray = new Local<Value>[length];
			for (size_t i = 0; i < length; i++)
			{
				char* c = result->at(i);
				valueArray[i] = ToJSON(isolate, ctx, c);
				delete[] c;
			}
			delete result;

			Local<Value> argv[1]{ Array::New(isolate, valueArray, length) };

			delete[] valueArray;
			Local<Function>::New(isolate, work->callback)->Call(ctx, ctx->Global(), 1, argv);
		}

		static void close_callback_func(uv_handle_t* request) {
			ListenerByteWork* work = static_cast<ListenerByteWork*>(request->data);
			delete work;
		}

		// For test purpose
		uint64_t now = uv_hrtime();
		void update_and_print(int tag) {
			uint64_t next = uv_hrtime();
			uint64_t duration = next - now;
			std::cout << tag << ":" << duration << std::endl;
			now = uv_hrtime();
		}
	};

	void init(Local<Object> exports, Local<Value> module, Local<Context> context) {
		Isolate* isolate = context->GetIsolate();

		// Client, on constructor
		auto Client = FunctionTemplate::New(isolate, V8Callback<_login, Pattern::CONSTRUCTOR, int64_t, CharUtil>);
		auto ClientString = String::NewFromUtf8(isolate, "Client").ToLocalChecked();
		Client->SetClassName(ClientString);

		// Client, on instance
		auto inst_t = Client->InstanceTemplate();
		inst_t->SetInternalFieldCount(1);

		// Client, on prototype
		auto proto_t = Client->PrototypeTemplate();
		proto_t->Set(v8::Symbol::GetToStringTag(isolate), ClientString, static_cast<v8::PropertyAttribute>(v8::ReadOnly | v8::DontEnum | v8::DontDelete));
		AddMethod<_onPrivateMessage, Pattern::INSTANCE_LISTENER>(isolate, proto_t, "onPrivateMessage");
		AddMethod<getFriendList, Pattern::INSTANCE_SYNC>(isolate, proto_t, "getFriendList");
		AddMethod<getGroupList, Pattern::INSTANCE_SYNC>(isolate, proto_t, "getGroupList");
		AddMethod<getGroupInfo, Pattern::INSTANCE_SYNC, int64_t>(isolate, proto_t, "getGroupInfo");
		AddMethod<getGroupMemberList, Pattern::INSTANCE_SYNC, int64_t>(isolate, proto_t, "getGroupMemberList");

		exports->Set(
			context,
			ClientString,
			Client->GetFunction(context).ToLocalChecked()
		);
	}
}

NODE_MODULE_INIT(/*exports, module, context*/) {
	ATRI::init(exports, module, context);
}
