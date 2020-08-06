#include "calculate_pi.h"
#include <node.h>

namespace calc {

  using v8::FunctionCallbackInfo;
  using v8::Isolate;
  using v8::Local;
  using v8::Object;
  using v8::String;
  using v8::Value;
  using v8::Number;
  using v8::Exception;
  using v8::Context;
  using v8::Function;
  using v8::FunctionCallbackInfo;
  using v8::FunctionTemplate;
  using v8::Isolate;
  using v8::Local;
  using v8::MaybeLocal;
  using v8::Number;
  using v8::Object;
  using v8::ObjectTemplate;
  using v8::String;
  using v8::Value;


  void calculate_pi(const FunctionCallbackInfo<Value>& args) {
    Isolate* isolate = args.GetIsolate();
    Local<Context> context = isolate->GetCurrentContext();

    // Check the number of arguments passed.
    // if (args.Length() < 2) {
    //   // Throw an Error that is passed back to JavaScript
    //   isolate->ThrowException(Exception::TypeError(
    //       String::NewFromUtf8(isolate, "Wrong number of arguments")));
    //   return;
    // }

    // // Check the argument types
    // if (!args[0]->IsNumber() || !args[1]->IsNumber()) {
    //   isolate->ThrowException(Exception::TypeError(
    //       String::NewFromUtf8(isolate, "Wrong arguments")));
    //   return;
    // }

    // Perform the operation
    char* str = CalculatePI(args[0]->Uint32Value(context).FromMaybe(0));
    Local<String> ret = String::NewFromOneByte(isolate, (uint8_t*)str).ToLocalChecked();

    // Set the return value (using the passed in
    // FunctionCallbackInfo<Value>&)
    args.GetReturnValue().Set(ret);
  }

  void init(Local<Object> exports) {
    NODE_SET_METHOD(exports, "calculate_pi", calculate_pi);
  }

  NODE_MODULE(calculator, init)
}
