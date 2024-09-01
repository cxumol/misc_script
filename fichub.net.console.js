function generateWgetFichub(url) {
  const givenFileName = new URL(url).pathname.split('/').pop();
  const baseFileName = givenFileName.split('-').slice(0, -1).join('-') + '.epub';
  return `wget "${url}" -O "${baseFileName}"`;
}

var url = '' // Download as EPUB @ https://fichub.net/
copy(generateWgetFichub(url))

/* no open tab */

async function fichubToWget(url) {
    const data = await fetch(`https://fichub.net/api/v0/epub?q=${encodeURIComponent(url)}`).then(r=>r.json());
    const fullEpubUrl = `https://fichub.net${data.epub_url}`;
    const baseFileName = new URL(fullEpubUrl).pathname.split('/').pop().split('-').slice(0, -1).join('-') + '.epub';
    const ans=`wget "${fullEpubUrl}" -O "${baseFileName}"`;
    return ans;
}
var url = '';
fichubToWget(url).then(cmd => copy(cmd));
