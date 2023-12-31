## chapters

> POST https://meeting.tencent.com/wemeet-tapi/v2/meetlog/public/record-detail/get-chapter

1. [jq](https://www.devtoolsdaily.com/jq_playground/) `.chapter_list | map({"title","start_time"})`
2. js console

```js
var data = [];
(function() {
function msToHms(ms) {
    if (typeof ms === 'string') {
        ms = parseInt(ms, 10);
    }
    const date = new Date(ms);
    return date.toISOString().substr(11, 8);
}

const result =  data.map(item => {
  return `${msToHms(item.start_time)} ${item.title}`
}).join('\n')
console.log(result)
})(data);
```
