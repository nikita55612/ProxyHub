#!/bin/bash
set -e

[[ $EUID -eq 0 ]] || exec sudo "$0" "$@"

PROXY_HUB_INSTALL_DIR="/opt/proxyhub"

inputv() {
    local v=""
    while [[ -z "$v" ]]; do
        read -p "$1" v
    done
    echo "$v"
}

validateport() {
    [[ "$1" =~ ^[0-9]+$ ]] && [ "$1" -ge 1024 ] && [ "$1" -le 65535 ]
}

inputport() {
    local port=""
    while ! validateport "$port"; do
        read -p "$1" port
    done
    echo "$port"
}

proxyhubparams=""

proxyhubmod=""
while ! [[ "$proxyhubmod" == "1" || "$proxyhubmod" == "2" ]]; do
    proxyhubmod=$(inputv "Введите режим ProxyHub (1 - infoserver, 2 - infoserver + server + telebot): ")
done

if [ "$proxyhubmod" = "1" ]; then
    rm -rf assets
    infoserverport=$(inputport "Введите порт для infoserver (1024-65535): ")
    proxyhubparams="-mode=1 -iport=$infoserverport"
else
    infoserverport=$(inputport "Введите порт для infoserver (1024-65535): ")
    serverport=$(inputport "Введите порт для server (1024-65535): ")
    genprefix=$(openssl rand -hex 16)
    proxyhubparams="-mode=2 -iport=$infoserverport -port=$serverport -prefix=/$genprefix"

    telebottoken=$(inputv "Введите токен телеграм бота: ")
    telebotownerid=$(inputv "Введите ID владельца телеграм бота: ")
    telebotaccesscode=$(inputv "Введите код доступа к телеграм боту: ")
    donutlink=$(inputv "Введите ссылку на пожертвования: ")
    telebotlink=$(inputv "Введите ссылку на телеграм бота: ")
    telebotlink="$telebotlink?start=$telebotaccesscode"

    cat > .env <<EOF
TELEGRAM_BOT_TOKEN=$telebottoken
TELEGRAM_BOT_OWNER_ID=$telebotownerid
TELEGRAM_BOT_ACCESS_CODE=$telebotaccesscode
PUBVAR_DONUT_LINK=$donutlink
PUBVAR_TELEGRAM_BOT_LINK=$telebotlink
EOF

    cat > proxyservers.json <<EOF
[
    {
        "name": "",
        "id": "",
        "location": "",
        "providerName": "",
        "providerLink": "",
        "plan": "",
        "speedRate": "",
        "limit": "",
        "infoLink": "",
        "proxyLinks": {
            "vless": [],
            "http": [],
            "socks": []
        }
    }
]
EOF
	echo "Необходимо внести информацию о proxy серверах в файл $PROXY_HUB_INSTALL_DIR/proxyservers.json"
fi

go build -o proxyhub

mkdir -p "$PROXY_HUB_INSTALL_DIR"
shopt -s dotglob nullglob
mv * "$PROXY_HUB_INSTALL_DIR"/

if ! systemctl list-unit-files --type=service | grep -q proxyhub.service; then
    cat > /etc/systemd/system/proxyhub.service <<EOF
[Unit]
Description=ProxyHub Server
After=network.target

[Service]
Type=simple
WorkingDirectory=$PROXY_HUB_INSTALL_DIR
ExecStart=$PROXY_HUB_INSTALL_DIR/proxyhub $proxyhubparams
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable proxyhub.service
fi

systemctl restart proxyhub.service || systemctl start proxyhub.service
