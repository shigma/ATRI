#include <node.h>
#include <uv.h>
#include <string.h>
#include <fstream>
#include <ios>
#include <functional>
#include <iostream>
#include "../src/main.h"

#define ENSURE_UV(x) assert(x == 0);

namespace ATRI {
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

	/* Multi-instance test, not used yet */
	// class AddonContext
	// {
	// public:
	// 	Local<ObjectTemplate> tpl;
	// public:
	// 	AddonContext(Isolate* isolate): tpl(ObjectTemplate::New(isolate)) {
	// 		node::AddEnvironmentCleanupHook(isolate, Dispose, this);
	// 		tpl->SetAccessor(String::NewFromUtf8(isolate, "t1").ToLocalChecked(), GetPath);
	// 	}
	// 	static void GetPath(Local<String> name, const PropertyCallbackInfo<Value>& info) {
	// 		RequestData* request = UnwrapRequest(info.Holder());
	// 		const char* path = request->b;
	// 		info.GetReturnValue().Set(String::NewFromUtf8(info.GetIsolate(), path).ToLocalChecked());
	// 	}
	// 	static RequestData* UnwrapRequest(Local<Object> obj) {
	// 		Local<External> field = Local<External>::Cast(obj->GetInternalField(0));
	// 		void* ptr = field->Value();
	// 		return static_cast<RequestData*>(ptr);
	// 	}
	// 	~AddonContext() {
	// 	}
	// 	static void Dispose(void* arg) {
	// 		delete static_cast<AddonContext*>(arg);
	// 	}
	// private:
	// };

	const std::string ToString(Isolate* isolate, Local<String> str) {
		String::Utf8Value value(isolate, str);
		return *value ? *value : "<string conversion failed>";
	}

	struct Work {
		uv_async_t request{};

		std::string url;
		char* result = nullptr;

		v8::Persistent<Function> node_callback;
		static void go_callback_func(uintptr_t ctx, char* ret) {
			Work* work = reinterpret_cast<Work*>(ctx);
			work->result = ret;
			ENSURE_UV(uv_async_send(&work->request));
		}
		static void node_callback_func(uv_async_t* request) {
			Work* work = static_cast<Work*>(request->data);

			Isolate* isolate = Isolate::GetCurrent();
			v8::HandleScope handleScope(isolate);

			Local<Context> ctx = isolate->GetCurrentContext();
			Local<Value> argv[2] = { v8::Null(isolate), String::NewFromUtf8(isolate, work->result).ToLocalChecked() };
			Local<Function>::New(isolate, work->node_callback)->Call(ctx, ctx->Global(), 2, argv);
			work->node_callback.Reset();
			uv_close((uv_handle_t*)request, NULL);
		}
		static void close_callback_func(uv_handle_t* request) {
			Work* work = static_cast<Work*>(request->data);
			delete work;
		}
		~Work()
		{
			// delete result;
		}
	};

	void login(const FunctionCallbackInfo<Value>& args) {
		Isolate* isolate = args.GetIsolate();
		Local<Context> ctx = isolate->GetCurrentContext();

		const int64_t uid = args[0].As<Integer>()->Value();
		const std::string psw = ToString(isolate, Local<String>::Cast(args[1]));
		_login(uid, const_cast<char*>(psw.c_str()));
	}

	void onPrivateMessage(const FunctionCallbackInfo<Value>& args) {
		Isolate* isolate = args.GetIsolate();
		Local<Context> ctx = isolate->GetCurrentContext();

		const std::string url = ToString(isolate, Local<String>::Cast(args[0]));
		const Local<Function> callback = Local<Function>::Cast(args[1]);

		Work* work = new Work;
		work->request.data = work;
		work->request.close_cb = work->close_callback_func;
		work->node_callback.Reset(isolate, callback);
		work->url = url;

		ENSURE_UV(uv_async_init(uv_default_loop(), &work->request, work->node_callback_func));
		_onPrivateMessage(const_cast<char*>(url.c_str()), reinterpret_cast<size_t>(work), work->go_callback_func);

		args.GetReturnValue().Set(Undefined(isolate));
	}

	// v8::ObjectTemplate tpl;
	// void test(const FunctionCallbackInfo<Value>& args) {
	// 	Isolate* isolate = args.GetIsolate();
	// 	Local<Context> ctx = isolate->GetCurrentContext();
	// 	AddonContext* addon = static_cast<AddonContext*>(args.Data().As<External>()->Value());
	// 	Local<Object> obj = addon->tpl->NewInstance(ctx).ToLocalChecked();
	// }

	void init(Local<Object> exports, Local<Value> module, Local<Context> context) {
		Isolate* isolate = context->GetIsolate();
		// AddonContext* addon = new AddonContext(isolate);
		// Local<External> external = External::New(isolate, addon);

		exports->Set(
			context,
			String::NewFromUtf8(isolate, "login").ToLocalChecked(),
			FunctionTemplate::New(isolate, login)->GetFunction(context).ToLocalChecked()
		);

		exports->Set(
			context,
			String::NewFromUtf8(isolate, "onPrivateMessage").ToLocalChecked(),
			FunctionTemplate::New(isolate, onPrivateMessage)->GetFunction(context).ToLocalChecked()
		);
	}
}

NODE_MODULE_INIT(/*exports, module, context*/) {
	ATRI::init(exports, module, context);
}
