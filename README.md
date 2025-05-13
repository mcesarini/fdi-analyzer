# fdi-analyzer
Golang program to analyze .fdi files
```
go build fdi_analyzer.go
Basic file inspection: ./fdi_analyzer -file your_file.fdi
View specific section: ./fdi_analyzer -file your_file.fdi -offset 1024 -bytes 512
Search for text: ./fdi_analyzer -file your_file.fdi -search "JUVENTUS"
```