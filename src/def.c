#include "def.h"

void InvokeCallback(Callback cb, uintptr_t ctx, char* arg) {
    cb(ctx, arg);
}
