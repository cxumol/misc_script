### rename all pngs into %04d.png


```ps1
$files = Get-ChildItem -Filter "*.png" | Sort-Object

for ($i = 0; $i -lt $files.Count; $i++) {
    $newName = "{0:D4}.png" -f ($i + 1)
    Rename-Item -Path $files[$i].FullName -NewName $newName
}
```

