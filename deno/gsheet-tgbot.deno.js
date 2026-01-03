/* for only data entry for now
tg bot setup on @botfather; google auth setup following https://developers.google.com/workspace/sheets/api/scopes#configure-oauth;
google auth notes: 
Set https://developers.google.com/oauthplayground as the API client callbak URI in order to manually get access_token & refresh_token; 
Enable Google sheet api at https://console.developers.google.com/apis/api/sheets.googleapis.com/overview
For accessing gdrive file with `https://www.googleapis.com/auth/drive.file`, the file has to be created by the same API client, e.g.
curl -X POST -H "Authorization: Bearer YOUR_ACCESS_TOKEN" -H "Content-Type: application/json" -d '{"properties":{"title":"temp- gsheet api test with scoped api access"}}' "https://sheets.googleapis.com/v4/spreadsheets"
*/

import {Bot,webhookCallback,InlineKeyboard} from "https://deno.land/x/grammy@v1.39.2/mod.ts";
// var TG={tok:"123:ABC",white:[123456],mode:"HTML"};
// var G={id:"sheets_id",ref:"oauth_refresh",cid:"client_id",csec:"client_sec"};
// var AI={
//     base:"https://api.example.com/v1",key:"sk-...",
//     chat:{mod:"gpt-4o"},
//     asr:{url:"https://cexample.ai/api/asr-inference"},
//     ocr:{url:"https://example.hf.space/run/predict",lang:"en"}
// };
//Help
var echo=s=>console.log(s),trim=s=>s.trim(),btwn=(s,b,e)=>{let i=s.indexOf(b),j=s.lastIndexOf(e);return(i<0||j<0||i>=j)?"":s.slice(i+b.length,j)},b64=b=>new Promise(r=>{var f=new FileReader();f.onload=()=>r(f.result.split(',')[1]);f.readAsDataURL(b)});
var api=async(u,o)=>await fetch(u,o).then(r=>r.json()),asmsg=(s,u)=>[{role:"system",content:s},{role:"user",content:u}]; //don't assume chat api provider supports json mode
//Adapters
var ADPT={oai:(ep,k,m,b)=>api(ep+"/chat/completions",{method:"POST",headers:{Authorization:"Bearer "+k,"Content-Type":"application/json"},body:JSON.stringify({model:m,messages:b})}).then(v=>v.choices[0].message.content),
    gradio:(ep,d)=>api(ep,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({data:d})}).then(v=>v.data),
    asr:(ep,d)=>api(ep,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({audio_file:{data:d,name:"a.ogg",type:"audio/ogg",size:d.length},language:"auto"})}).then(v=>v.data[0])
};
var RUN={CHAT:m=>ADPT.oai(AI.base,AI.key,AI.chat.mod,m),
    ASR:d=>ADPT.asr(AI.asr.url,d), OCR:d=>ADPT.gradio(AI.ocr.url,[`data:image/jpeg;base64,${d}`,AI.ocr.lang]).then(r=>r[0])
};
//Core
var gTok=async()=>((await api("https://oauth2.googleapis.com/token",{method:"POST",body:JSON.stringify({client_id:G.cid,client_secret:G.csec,refresh_token:G.ref,grant_type:"refresh_token"})})).access_token);
var proc=async(ctx,txt)=>{echo({userinput: txt});
    try{var t=await gTok(),url=(r,x="")=>`https://sheets.googleapis.com/v4/spreadsheets/${G.id}/values/${encodeURIComponent(r)}${x}`,auth={headers:{Authorization:"Bearer "+t}};
        //echo(t);
        var r1=await api(url("Sheet1!1:1"),auth); if(r1.error)throw new Error(r1.error.message);
        var heads=r1.values?.[0]||[];if(!heads.length)return ctx.reply("Headers empty.");
        var p=`Parse input to cols: ${heads.join(',')}. Fill empty/missing value as "N/A". Input: ${txt}. Output JSON: \`\`\`json\n{"${heads[0]}":"val",...}\n\`\`\``;
        var raw=await RUN.CHAT(asmsg("Entry Bot",p)),json=JSON.parse(btwn(raw,"```json","```").trim()||raw);
        var res=await api(url("Sheet1!A:A",":append?valueInputOption=USER_ENTERED"),{method:"POST",headers:{...auth.headers,"Content-Type":"application/json"},body:JSON.stringify({values:[heads.map(h=>json[h]||"")]})});
        if(res.error)throw new Error(res.error.message);
        var rng=res.updates.updatedRange,tmsg=`<b>Added:</b>\n<pre>${JSON.stringify(json,null,1)}</pre>\n<code>https://docs.google.com/spreadsheets/d/${G.id}/</code>`;
        return ctx.reply(tmsg,{parse_mode:TG.mode,reply_markup:new InlineKeyboard().text("Retract 🗑️",`del:${rng}`)});
    }catch(e){console.error(e);return ctx.reply("Err: "+e.message)}
};
//Bot
var bot=new Bot(TG.tok);
bot.use(async(c,n)=>{if(!TG.white.includes(c.chat?.id))return await c.reply(`chatid ${c.chat?.id} not in whitelist`);await n()});
bot.on("message:text",c=>proc(c,c.msg.text));
bot.on(["message:voice","message:photo"],async c=>{var isA=!!c.msg.voice,fid=isA?c.msg.voice.file_id:c.msg.photo.slice(-1)[0].file_id;
    var f=await c.api.getFile(fid),u=`https://api.telegram.org/file/bot${TG.tok}/${f.file_path}`;
    var b=await fetch(u).then(r=>r.blob()),d=await b64(b),txt=isA?await RUN.ASR(d):await RUN.OCR(d);
    await c.reply(`🖼️/🎤: ${txt}`); return proc(c,txt);
});
bot.on("callback_query:data",async c=>{if(!c.callbackQuery.data.startsWith("del:"))return;
    var r=c.callbackQuery.data.slice(4),t=await gTok();
    await api(`https://sheets.googleapis.com/v4/spreadsheets/${G.id}/values/${encodeURIComponent(r)}:clear`,{method:"POST",headers:{Authorization:"Bearer "+t}});
    await c.editMessageText(`🗑️ Retracted ${r}`);
});
//Server
var h=webhookCallback(bot,"std/http");
Deno.serve(async e=>{if(e.method==="POST"&&new URL(e.url).pathname.slice(1)===bot.token)try{return await h(e)}catch(n){console.error(n)}return new Response});
