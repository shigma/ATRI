# ATRI

アトリは、高性能ですから!

```bat
ts-node tools/build
node ./test.js
```

## roadmap
- [x] 前期调研，确认可行性
- [ ] 编写自动化脚本和工具
- [ ] 整理接口

## 基本思路
- 将Go方法最外层使用goroutine包装，将阻塞方法转化为callback型异步方法，通过`uv_async_init`和`uv_async_send`进行对接，使goroutine能够与uvlib协调使用3
  注：最初使用`uv_queue_work`，发现属于misuse，并发受限，并且阻塞worker pool
- 数据交换考虑使用 cap'n proto (或 protobuf等类似格式), 理由如下：
  - Go结构体无法自动向C结构体转换，手动转换，尤其在处理指针等问题上，存在一定工作量
  - C结构体无法自动向JS对象转换，需要使用v8::ObjectTemplate包装或者转换为v8::Object，有一定工作量，同时前者的getter/setter模型阻碍JIT内联代码，效率可能低于JS本身
  - 考虑使用cap'n proto序列化/反序列化数据，通过生成的代码直接读写`[]byte`(在JS中是`ArrayBuffer`), C++只负责中转内存buffer
  - 可以基于反射，基于现有Go结构体，比较方便地生成cap'n proto，见`tools/reflect.go`
  - cap'n proto可以自动生成TS类型
