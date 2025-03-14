var scheme_flypy_={id:"flypy",name:"小鹤双拼",tips:["iOS (>=12.1.1) 自带方案"],detail:{sheng:{b:"b",c:"c",d:"d",f:"f",g:"g",h:"h",j:"j",k:"k",l:"l",m:"m",n:"n",p:"p",q:"q",r:"r",s:"s",t:"t",w:"w",x:"x",y:"y",z:"z",ch:"i",sh:"u",zh:"v"},yun:{a:"a",ai:"d",an:"j",ang:"h",ao:"c",e:"e",ei:"w",en:"f",eng:"g",i:"i",ia:"x",ian:"m",iang:"l",iao:"n",ie:"p",iong:"s",in:"b",ing:"k",iu:"q",o:"o",ong:"s",ou:"z",u:"u",ua:"x",uai:"k",uan:"r",uang:"l",ue:"t",ui:"v",un:"y",uo:"o",v:"v",ve:"t"},other:{a:"aa",ai:"ai",an:"an",ang:"ah",ao:"ao",e:"ee",ei:"ei",en:"en",eng:"eg",er:"er",o:"oo",ou:"ou"}}}; // pinyin => flypy // ref: https://github.com/BlueSky-07/Shuang/blob/master/src/scheme/xiaohe.js

// reverse mapping from scheme_flypy_ ==> flypy => pinyin
var shengMap={},yunMap={},otherMap={};
for(let key in scheme_flypy_.detail.sheng) shengMap[scheme_flypy_.detail.sheng[key]]=key;
for(let key in scheme_flypy_.detail.yun) yunMap[scheme_flypy_.detail.yun[key]]=key;
for(let key in scheme_flypy_.detail.other) otherMap[scheme_flypy_.detail.other[key]]=key;

// ---------------

var shengMap={"b":"b","c":"c","d":"d","f":"f","g":"g","h":"h","j":"j","k":"k","l":"l","m":"m","n":"n","p":"p","q":"q","r":"r","s":"s","t":"t","w":"w","x":"x","y":"y","z":"z","i":"ch","u":"sh","v":"zh"};
var yunMap={"a":"a","d":"ai","j":"an","h":"ang","c":"ao","e":"e","w":"ei","f":"en","g":"eng","i":"i","x":"ua","m":"ian","l":"uang","n":"iao","p":"ie","s":"ong","b":"in","k":"uai","q":"iu","o":"uo","z":"ou","u":"u","r":"uan","t":"ve","v":"v","y":"un"};
var otherMap={"aa":"a","ai":"ai","an":"an","ah":"ang","ao":"ao","ee":"e","ei":"ei","en":"en","eg":"eng","er":"er","oo":"o","ou":"ou"};

// ----------------

function flypyToPinyin(flypyStr){
  var shengMap={"b":"b","c":"c","d":"d","f":"f","g":"g","h":"h","j":"j","k":"k","l":"l","m":"m","n":"n","p":"p","q":"q","r":"r","s":"s","t":"t","w":"w","x":"x","y":"y","z":"z","i":"ch","u":"sh","v":"zh"};
  var yunMap={"a":"a","d":"ai","j":"an","h":"ang","c":"ao","e":"e","w":"ei","f":"en","g":"eng","i":"i","x":"ua","m":"ian","l":"uang","n":"iao","p":"ie","s":"ong","b":"in","k":"uai","q":"iu","o":"uo","z":"ou","u":"u","r":"uan","t":"ve","v":"v","y":"un"};
  var otherMap={"aa":"a","ai":"ai","an":"an","ah":"ang","ao":"ao","ee":"e","ei":"ei","en":"en","eg":"eng","er":"er","oo":"o","ou":"ou"};
  var cleanedStr=flypyStr.replace(/[\s\p{P}]+/gu,'');
  var pinyin='', ending=' ';
  for(let i=0;i<cleanedStr.length;i+=2){
    const part=cleanedStr.substring(i,i+2);
    if(shengMap[part[0]] && yunMap[part[1]]){
      pinyin+=shengMap[part[0]]+yunMap[part[1]]+ending;
    }else if(otherMap[part]){
      pinyin+=otherMap[part]+ending;
    }
  }
  return pinyin;
}
