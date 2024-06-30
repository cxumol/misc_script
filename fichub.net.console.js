function generateWgetFichub(url) {
  const givenFileName = new URL(url).pathname.split('/').pop();
  const baseFileName = givenFileName.split('-').slice(0, -1).join('-') + '.epub';
  return `wget "${url}" -O "${baseFileName}"`;
}

var url = '' // Download as EPUB @ https://fichub.net/
copy(generateWgetFichub(url))
