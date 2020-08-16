# ATRI

アトリは、高性能ですから!

## Roadmap

- [x] 前期调研，确认可行性
- [ ] 确定接口，编写实现
  - [ ] 入口（构造函数 + 绑定回调 + 账号登录）
  - [ ] 事件（Go 包装成 JSON，通过单一回调函数进行传递）
  - [ ] API（Go 实现成回调函数，TS 中转化为 Promise）
- [ ] 完成全平台编译

## 基本思路
- 将Go方法最外层使用goroutine包装，将阻塞方法转化为callback型异步方法，通过`uv_async_init`和`uv_async_send`进行对接，使goroutine能够与uvlib协调使用3
  注：最初使用`uv_queue_work`，发现属于misuse，并发受限，并且阻塞worker pool
- 目前，JS向Go传递数据时，简单数据进行参数转换，复杂数据直接通过JSON序列化传递；返回值统一通过JSON序列化传递
