# [mc-server](https://hub.docker.com/r/aishukander/mc-server)

## 說明
基於debian製作的minecraft server，java和伺服器檔會在啟動後從網路下載，以減少映像檔的大小。 <br>
跟伺服器有關的檔案都在容器的 /project/server 目錄下。 <br>
跟java有關的檔案都在容器的 /project/java 目錄下。 <br>

## 支援度
如果遇到java版本不符的問題可以通過在compose的環境變數增加JAVA_VERSION_OVERRIDE來更改java版本。 <br>
支援的java版本可到連接查詢[releases](https://adoptium.net/temurin/releases/)
* [Paper](https://papermc.io/downloads/all) 列表中所有paper版本
* [NeoForge](https://projects.neoforged.net/neoforged/neoforge) 可選的所有neoforge版本
* [Other](https://mcutils.com/server-jars) Paper跟NeoForge以外都是基於mcutils，具體支援類型可到連結內查看(但不支援forge)。

## 伺服器後台
在宿主機使用```docker attach mc-server```進入伺服器後台，按下 Ctrl+P+Q 退出後台。 <br>

## 啟動
Docker compose <br>
```yml
services:
  mc-server:
    container_name: mc-server
    tty: true
    stdin_open: true
    image: aishukander/mc-server
    restart: on-failure
    environment:
      #JAVA_VERSION_OVERRIDE: "<version>"
      Type: "<paper/neoforge/other>"
      MINECRAFT_VERSION: "<minecraft_version>"
      Min_Ram: "<min_ram>"
      Max_Ram: "<max_ram>"
    volumes:
      - <host_data_path>:/project/server
    ports:
      - "<host_port>:25565"
```