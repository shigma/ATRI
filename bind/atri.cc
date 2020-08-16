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
#include "common.h"

#define ENSURE_UV(x) assert(x == 0);

namespace ATRI {
	void init(Local<Object> exports, Local<Value> module, Local<Context> context) {
		Isolate* isolate = context->GetIsolate();

		// Client, on constructor
		auto Client = FunctionTemplate::New(isolate, V8Callback<_newCQBot, Pattern::CONSTRUCTOR, int64_t, CharUtil>);
		auto ClientString = String::NewFromUtf8(isolate, "Client").ToLocalChecked();
		Client->SetClassName(ClientString);
		AddMethod<_setupDevice, Pattern::PLAIN, JsonUtil>(isolate, Client, "setupDevice");

		// Client, on instance
		auto inst_t = Client->InstanceTemplate();
		inst_t->SetInternalFieldCount(1);

		// Client, on prototype
		auto proto_t = Client->PrototypeTemplate();
		proto_t->Set(v8::Symbol::GetToStringTag(isolate), ClientString, static_cast<v8::PropertyAttribute>(v8::ReadOnly | v8::DontEnum | v8::DontDelete));
		AddMethod<_login, Pattern::INSTANCE_ASYNC>(isolate, proto_t, "login");
		AddMethod<_loginInteractive, Pattern::INSTANCE_SYNC>(isolate, proto_t, "loginSync");
		AddMethod<_sendPrivateMessage, Pattern::INSTANCE_ASYNC, int64_t, CharUtil>(isolate, proto_t, "sendPrivateMessage");
		AddMethod<_sendGroupMessage, Pattern::INSTANCE_ASYNC, int64_t, CharUtil>(isolate, proto_t, "sendGroupMessage");
		AddMethod<onEvent, Pattern::INSTANCE_LISTENER>(isolate, proto_t, "onEvent");
		AddMethod<getFriendList, Pattern::INSTANCE_SYNC>(isolate, proto_t, "getFriendList");
		AddMethod<getGroupList, Pattern::INSTANCE_SYNC>(isolate, proto_t, "getGroupList");
		AddMethod<getGroupInfo, Pattern::INSTANCE_SYNC, int64_t>(isolate, proto_t, "getGroupInfo");
		AddMethod<getGroupMemberList, Pattern::INSTANCE_SYNC, int64_t>(isolate, proto_t, "getGroupMemberList");
		
		WRAP_UNUSED(exports->Set(
			context,
			ClientString,
			Client->GetFunction(context).ToLocalChecked()
		));
	}
}

NODE_MODULE_INIT(/*exports, module, context*/) {
	ATRI::init(exports, module, context);
}
