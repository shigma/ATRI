// test.js
const { request } = require('./sample/binding/build/Release/atri');

request("http://www.baidu.com", (err, body) => {
    console.log(body.slice(0, 1000))
})
