import * as cheerio from "npm:cheerio"; import { Readability } from "jsr:@paoramen/cheer-reader";
import { parse as parseTph, Telegraph } from "jsr:@dcdunkan/telegraph";
var mdToNode=(txt)=>parseTph(txt,"Markdown");
var tphTk = ["abcdef123456"]; // https://telegra.ph/api#createAccount
var sk='key1,key2,key3'; // without sk-
var cfg={"prod1":{"base":"https://api.example.com/v1","model":"gpt-100","k":sk.split(',').map(x=>'sk-'+x)},
"prod2":{"base":"https://api.example.org/v1","model":"DeepSeek-V587","k":["sk-123"]},
};
var sys={"提取正文":'Extract main article boundaries from text for Mozilla Pocket Reader View. Find the minimal length of first phrase and the last phrase that can identify article boundaries, excluding non-article sections (nav, h1, author list, related articles, comments, etc.) Output ONLY JSON: ```json\n{"mainArticleContent":{"startsWith":"<string>","endsWith":"<string>"}}\n```',
"笔记诗":"仔细研读全文后写成中文笔记,以供传阅及备忘｡笔记应如庖丁解牛,切中肯絮,鞭辟入里;行文干净流畅,如史蒂芬品克《风格感觉》之素雅;词句考究,似侯世达、严勇、刘皓明、莫大伟《集异璧之大成中文版》之精妙｡另以全文内容为题赋诗,以添文韵,笔记开头加上五言绝句来定场,结尾附上七言律诗以收场｡",
};
var cfg_mapping={"提取正文":"prod1","笔记诗":"prod2"};
var rand = (l=cfg.prod1.k.length, i=0) => l===1 ? 0 : (i + parseInt(performance.now())%20) % l;
var oai=async(api,msg,t=0.6)=>await fetch(api.base+'/chat/completions',{method:'POST',headers:{Authorization:'Bearer '+api.key,'Content-Type':'application/json'},body:JSON.stringify({model:api.model,messages:msg,temperature:t})}).then(r=>r.text()).then(j=>{let m;try{m=JSON.parse(j).choices?.[0]?.message}catch{throw("JSON.parse err\n"+j)}if(!m?.content)throw(j);let c=m.content.trim(),r=m.reasoning_content;return r?`<think>${r.trim()}</think>${c}`:c});
var retry_oai=async(_cfg,_i,msg,t=0.6)=>{for(let i=0;i<3;i++) try{return await oai(asapi(_cfg,_i+i),msg,t);}catch(err){console.error(err,msg);await new Promise(r=>setTimeout(r, 300)); continue;} }; //let ans=await oai(api,msg,temperature);console.log(ans);return ans;
var asapi=(_c,i=0)=>{return {..._c,'k':null,'key':_c.k[rand(_c.k.length,i)]} };
var asmsg=(_sys,_txt)=>[{"role":"system","content":_sys},{"role":"user","content":_txt}];
var btwn=(s,b,e)=>{var i=s.indexOf(b),j=s.lastIndexOf(e); if(i===-1||j===-1||i>=j)throw new Error(`btwn: start or end not found with<begin>${b}<end>${e}`);return s.substring(i, j+e.length);}
async function extractMainArticleContent(md_){ for(let i=0;i<3;i++){ try{
    var r = await retry_oai(cfg[cfg_mapping["提取正文"]], i, asmsg(sys["提取正文"], md_), 1.0);
    var jstr = btwn(r, '{', '}'); if (!jstr) throw new Error(`JSON not found in agent resp: ${r}`);
    var { startsWith, endsWith } = JSON.parse(jstr).mainArticleContent;
    ;console.log(`提取正文agent: ${startsWith} ... ${endsWith}`);
    return btwn(md_, startsWith.trim(), endsWith.trim());
}catch(er){ console.error(`Attempt ${i + 1} failed with 提取正文\n${r}\n`,er);if(i===2)throw new Error(`Failed to extract main article content after 3 attempts. ${er}`);
    await new Promise(r => setTimeout(r, 300));}
}}
async function url2md2llm(url) { var txt,md,title,parseMode;
  try{txt = await fetch(`https://r.jina.ai/`+url).then(d=>d.text());}catch(err){console.error(err);}; // {headers:{"X-Engine":"browser"}}
  if(!txt || txt.indexOf(": DDoS attack suspected:")!==-1 || txt.indexOf("Just a moment...")!==-1 )parseMode="moz"; //throw new Error("文章提取失败");
  if (!parseMode){
    try{md = txt.split("Markdown Content:")[1].trim(), title = txt.match(/Title: (.*?)\n/)?.[1];
    md = await extractMainArticleContent(md);}catch(e){console.error(e,txt);}
    if(!md?.length || md?.length<100)parseMode="moz";
  }
  if (parseMode=="moz"){ console.log("parseMode exp moz");
    txt = await fetch(url,{headers:{'User-Agent':"facebookexternalhit/1.1"}}).then(d=>d.text());
    const doc = new Readability(cheerio.load(txt)).parse();
    title = doc.title; md = doc.textContent; if(!md)throw new Error("文章提取失败 "+title);
  }
  md = md.replaceAll(/\n\n([^\n]+)\n-{3,}\n\n/g, (match, group1) => `\n\n## ${group1}\n\n` //将 Setext 风格的 H2 标题转换为 ATX 风格
        ).replaceAll(/([^\n]+)\n={3,}\n\n/g, (match, group1)=>`# ${group1}\n\n` //将 Setext 风格的 H1 标题转换为 ATX 风格
        ).replaceAll(/^[\t\f\v \u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000\ufeff]+$/gm,'\n\n' //空白行替换为标准的段落分隔符 `\n\n`
        ).replaceAll(/\n{3,}/g,'\n\n').replaceAll(/\[\!\[(.*?)\]\((.*?)\)\]\((.*?)\)/gm, '![$1]($2)' //将三个或更多的连续换行符压缩为两个 | 提取出内部的图片语法 `![alt](img.png)`，丢弃外部的链接
        ).replaceAll(/```(\w*)\n(.*?)```/gs, (match, lang, content) => `\`\`\`${lang}\n${content.replaceAll(/(\n\s*){2,}/g, '\n')}\`\`\``; //移除代码块内多余空行
        );;
  var rtxt = await retry_oai(cfg[cfg_mapping["笔记诗"]], 0, asmsg(sys["笔记诗"], md), 1.0);
  if(rtxt.includes("</think>")){var quot='';
      try{quot=btwn(rtxt,"<think>","</think>").slice(7,-8);quot=quot.replaceAll(/\n\s*\n+/g, '\n').split('\n').filter(l=>l.trim()!=='').map(l=>'> '+l).join('  \n');}catch(e){console.error(e)}
      rtxt=quot + rtxt.split("</think>")[1];
  }
  title = "Notes on "+title;
  return [rtxt, title];
}
async function md2Tph(md,ttl,url){
    var tph = new Telegraph({ token: tphTk[0] });tph.catch(e=>{throw e});
    var page = await tph.create({ title: ttl.slice(0,98), content: mdToNode(md), author_name: "阅读原文", author_url: url });
    ;console.log(page);
    return page.url;
}
async function main(req){var ingest,ingestUrl;
  ingest = new URL(req.url).searchParams.get("ingest");
  try {ingestUrl = new URL(ingest);}catch{return new Response("",{status:404});}
  var [rtxt, title] = await url2md2llm(ingestUrl);
  console.log(`${title}: notes len=${rtxt.length}`);
  return new Response(await md2Tph(rtxt,title,ingestUrl));
}
Deno.serve(main);
