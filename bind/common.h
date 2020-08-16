#ifndef V8_ATRI_BINDING_COMMON_H
#define V8_ATRI_BINDING_COMMON_H
#include <node.h>
#include <uv.h>
#include <string.h>
#include <fstream>
#include <ios>
#include <functional>
#include <iostream>
#include <atomic>
#include <stdexcept>
#define ENSURE_UV(x) assert(x == 0);

#define WRAP_UNUSED(expr) static_cast<void>(expr)

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
	using v8::Promise;
	using v8::PropertyCallbackInfo;
	using v8::String;
	using v8::Value;

	// Utility for v8::String to char* convertion
	struct CharUtil
	{
		String::Utf8Value* str;
		CharUtil(Isolate* isolate, Local<Value> value) : str(new String::Utf8Value(isolate, value)) {}
		operator char* () { return **str; }
		~CharUtil() {
			delete str;
		}
	};

	// Utility for v8::Object to char* convertion
	struct JsonUtil
	{
		String::Utf8Value* str;
		JsonUtil(Isolate* isolate, Local<Value> value) : str(new String::Utf8Value(isolate, v8::JSON::Stringify(isolate->GetCurrentContext(), value).ToLocalChecked())) {}
		operator char* () { return **str; }
		~JsonUtil() {
			delete str;
		}
	};

	// Binding template pattern
	enum class Pattern
	{
		PLAIN, // TBD
		CONSTRUCTOR,
		INSTANCE_SYNC,
		INSTANCE_ASYNC,
		INSTANCE_LISTENER,
		DESTRUCTOR // TBD
	};

	template<auto ERR>
	inline void V8Error(Isolate* isolate, const char* message) {
		isolate->ThrowException(ERR(String::NewFromUtf8(isolate, message).ToLocalChecked()));
	}
	constexpr auto TypeError = V8Error<Exception::TypeError>;

	template<typename T>
	inline T Convert(Isolate* isolate, Local<Value>&& value) {
		if constexpr (std::is_same_v<T, CharUtil>) {
			if (!value->IsString()) throw std::invalid_argument("expect string");
			return { isolate, value };
		}
		else if constexpr (std::is_same_v<T, JsonUtil>) {
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
		v8::Persistent<Promise::Resolver> resolver;
		AsyncByteWork(Isolate* isolate, Local<Promise::Resolver> resolver) : resolver(isolate, resolver) {
			request.data = this;
			request.close_cb = close_callback_func;
		}

		~AsyncByteWork() {
			delete result;
			delete error;
		}

		template<auto F, typename... Ts>
		void Invoke(Ts&&... args) {
			ENSURE_UV(uv_async_init(uv_default_loop(), &this->request, this->node_callback_func));
			F(args..., go_callback_func, reinterpret_cast<uintptr_t>(this));
		}

		void Dispose() {
			this->resolver.Reset();
			uv_close((uv_handle_t*)&request, NULL);
		}

		char* error = nullptr;
		char* result = nullptr;
		size_t length = -1;
		std::atomic_flag called = ATOMIC_FLAG_INIT;
		static void go_callback_func(uintptr_t ctx, void* result, void* error, size_t length) {
			AsyncByteWork* work = reinterpret_cast<AsyncByteWork*>(ctx);

			assert(!work->called.test_and_set()); // ensure called only once

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

			bool successful;
			if (work->error == nullptr) {
				successful = Local<Promise::Resolver>::New(isolate, work->resolver)->Resolve(ctx, ToJSON(isolate, ctx, work->result)).FromJust();
			}
			else {
				successful = Local<Promise::Resolver>::New(isolate, work->resolver)->Reject(ctx, ToJSON(isolate, ctx, work->error)).FromJust();
			}
			assert(successful);
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
		ListenerByteWork(Isolate* isolate, Local<Function> callback) : callback(isolate, callback) {
			request.data = this;
			request.close_cb = close_callback_func;
		}

		~ListenerByteWork() {
			delete buffer;
		}

		template<auto F, typename... Ts>
		void Invoke(Ts&&... args) {
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

			WRAP_UNUSED(Local<Function>::New(isolate, work->callback)->Call(ctx, ctx->Global(), 1, argv));
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

	template<auto F, Pattern P, size_t N, typename... A, size_t... S>
	inline void V8CallbackNumbered(const FunctionCallbackInfo<Value>& args, std::index_sequence<S...>) {
		Isolate* isolate = args.GetIsolate();
		constexpr size_t REAL_N = P == Pattern::INSTANCE_LISTENER ? N + 1 : N;
		if (static_cast<unsigned>(args.Length()) < REAL_N) {
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
					const Local<Promise::Resolver> resolver = Promise::Resolver::New(ctx).ToLocalChecked();
					AsyncByteWork* work = new AsyncByteWork(isolate, resolver);
					work->Invoke<F>(bot, std::forward<A>(Convert<A>(isolate, args[S]))...);
					args.GetReturnValue().Set(resolver->GetPromise());
				}
				else if constexpr (P == Pattern::INSTANCE_LISTENER) {
					ListenerByteWork* work = new ListenerByteWork(isolate, Convert<Local<Function>>(isolate, args[N]));
					work->Invoke<F>(bot, std::forward<A>(Convert<A>(isolate, args[S]))...);
					args.GetReturnValue().Set(Undefined(isolate));
				}
				else {
					// NEVER
					// static_assert(false, "Unimplemented");
				}
			}
		}
		else if constexpr (P == Pattern::PLAIN) {
			char* result;
			try {
				result = F(std::forward<A>(Convert<A>(isolate, args[S]))...);
			}
			catch (std::invalid_argument& ex) {
				return TypeError(isolate, ex.what());
			}
			args.GetReturnValue().Set(ToJSON(isolate, isolate->GetCurrentContext(), result));
			GoFree(result);
		}
		else {
			// static_assert(false, "Unimplemented");
		}
	}

	template<auto F, Pattern P, typename... A>
	void V8Callback(const FunctionCallbackInfo<Value>& args) {
		constexpr size_t N = sizeof...(A);
		constexpr auto S = std::make_index_sequence<N>{};
		V8CallbackNumbered<F, P, N, A...>(args, S);
	}

	template<auto F, Pattern P, typename... A>
	inline void AddMethod(Isolate* isolate, Local<v8::Template> tpl, const char* name) {
		tpl->Set(isolate, name, FunctionTemplate::New(isolate, V8Callback<F, P, A...>));
	}
}

#endif
