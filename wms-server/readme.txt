Go 尚未安裝在此環境中。請先至 https://go.dev/dl/ 
下載並安裝 Go 1.21 以上版本（Windows 選 .msi 安裝檔）。

安裝後重新開啟 PowerShell，確認可執行 "go version"，然後：

cd "D:\利美\內政部無人機反制案\wms-server"
go mod tidy
go build -o wms-server.exe .\cmd\wms-server\

編譯成功後會產生 wms-server.exe，執行前先修改 wms-server.yaml 中的圖資路徑對應到實際的 MBTiles/XYZ/Shapefile 檔案位置即可啟動


-----------------------------------------
PS D:\利美\內政部無人機反制案\wms-server> go mod tidy
go: downloading modernc.org/sqlite v1.29.5
go: downloading gopkg.in/yaml.v3 v3.0.1
go: downloading gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405
go: downloading modernc.org/libc v1.41.0
go: downloading modernc.org/fileutil v1.3.0
go: downloading golang.org/x/sys v0.16.0
go: downloading github.com/mattn/go-sqlite3 v1.14.22
go: downloading modernc.org/mathutil v1.6.0
go: downloading modernc.org/gc/v3 v3.0.0-20240107210532-573471604cb6
go: downloading github.com/google/pprof v0.0.0-20221118152302-e6195bd50e26
go: downloading golang.org/x/tools v0.17.0
go: downloading github.com/hashicorp/golang-lru/v2 v2.0.7
go: downloading github.com/dustin/go-humanize v1.0.1
go: downloading modernc.org/token v1.1.0
go: downloading modernc.org/strutil v1.2.0
go: downloading github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec
go: downloading github.com/google/uuid v1.3.0
go: downloading github.com/mattn/go-isatty v0.0.16
go: downloading modernc.org/memory v1.7.2
go: downloading github.com/ncruces/go-strftime v0.1.9
go: downloading github.com/pmezard/go-difflib v1.0.0
go: downloading golang.org/x/mod v0.14.0

-------------------------------------------------
PS D:\利美\內政部無人機反制案\wms-server> go build -o wms-server.exe .\cmd\wms-server\

Go 已安裝但 PATH 未更新。請重新開啟 PowerShell 視窗後再試，或手動執行：

$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
cd "D:\利美\內政部無人機反制案\wms-server"
go build -o wms-server.exe .\cmd\wms-server\


--------------------------------------
>>每一次部署都要那麼麻煩嗎?

不會，編譯只需做一次。編譯產生的 wms-server.exe 是單一自包含執行檔，之後部署到其他 Windows 電腦只需要：

wms-server.exe
wms-server.yaml
data/              ← 圖資目錄
直接複製這三樣過去就能執行，目標機器不需要安裝 Go。例如建立一個 deploy.bat：

copy wms-server.exe \\目標電腦\共享資料夾\
copy wms-server.yaml \\目標電腦\共享資料夾\
xcopy /E data \\目標電腦\共享資料夾\data\
或者做成 zip 壓縮檔，解壓即用。
-------------------------------

D:\利美\內政部無人機反制案\wms-server\Map\TaiwanEMap6.mbtiles
