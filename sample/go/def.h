#ifndef GO_CGO_DEF_TEST_H
#define GO_CGO_DEF_TEST_H
#include <stddef.h>

typedef void (*Callback)(uintptr_t, char*);
void InvokeCallback(Callback, uintptr_t, char*);

#endif
