## util

- jq https://www.devtoolsdaily.com/jq_playground/

## chapters

> POST https://meeting.tencent.com/wemeet-tapi/v2/meetlog/public/record-detail/get-chapter

1. jq `.chapter_list | map({"title","start_time"})`
2. js console

```js
var data = [];
(function(data) {
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
## transcriptions

> POST https://meeting.tencent.com/wemeet-cloudrecording-webapi/v1/minutes/detail  
> url query 的 limit 改大, 如 200

deno

```js

async function main(inputFile="tmpdata.json", outputFile="transcription.txt") {
  try {
    // Use Deno.readTextFile to read the JSON file
    const data = await Deno.readTextFile(inputFile);
    const parsedData = JSON.parse(data);

    const extracted = extract(parsedData);
    const outputText = extracted.map((e) => `${msToHms(e.start_time)} ${e.speaker}\n${e.text}`).join("\n\n");

    // Use Deno.writeTextFile to write the output to a text file
    await Deno.writeTextFile(outputFile, outputText);
    console.log("Transcription written to transcription.txt");
  } catch (error) {
    console.error("Error processing data:", error);
  }
}

function msToHms(ms) {
 if (typeof ms === "string") {
   ms = parseInt(ms, 10);
 }
 const date = new Date(ms);
 return date.toISOString().substr(11, 8);
}

function extract(data) {
 const paragraphs = data.paragraphs;

 const output = [];
 for (const paragraph of paragraphs) {
   const text = paragraph.sentences.reduce((acc = "", sentence) => {
     return acc + sentence.words.map((w) => w.text).join(" ");
   }, "");
   output.push({
     speaker: paragraph.speaker.user_name,
     text,
     start_time: paragraph.start_time,
   });
 }
 return output;
}

main();

```

or browser (crashy)

```js
var data = {};
(function(data) {
    function msToHms(ms) {
        if (typeof ms === 'string') {
            ms = parseInt(ms, 10);
        }
        const date = new Date(ms);
        return date.toISOString().substr(11, 8);
    }
    
    function extract(){
        const paragraphs = data.paragraphs;
        
        const output = [];
        for (const paragraph of paragraphs) {
          const text = paragraph.sentences.reduce((acc= "" , sentence) => {
            return acc + sentence.words.map(w => w.text).join(" ")
          }, "");
          output.push({
            speaker: paragraph.speaker.user_name,
            text,
            start_time: paragraph.start_time,
          });
        }
        return output;
    }
    const extracted = extract();
    const output_text = extracted.map(e=>`${msToHms(e.start_time)} ${e.speaker}\n${e.text}`).join(`\n\n`)
    console.log(output_text)
})(data);

```
