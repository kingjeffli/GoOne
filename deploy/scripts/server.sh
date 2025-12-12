#!/bin/bash

#ulimit -c unlimited
source /etc/profile

# 颜色支持（如果是非 TTY 或设置了 NO_COLOR，则自动关闭颜色）
if [[ -t 1 && -z "${NO_COLOR:-}" ]]; then
    COLOR_RED=$'\033[0;31m'
    COLOR_GREEN=$'\033[0;32m'
    COLOR_YELLOW=$'\033[1;33m'
    COLOR_BLUE=$'\033[0;34m'
    COLOR_CYAN=$'\033[0;36m'
    COLOR_BOLD=$'\033[1m'
    COLOR_RESET=$'\033[0m'
else
    COLOR_RED=""
    COLOR_GREEN=""
    COLOR_YELLOW=""
    COLOR_BLUE=""
    COLOR_CYAN=""
    COLOR_BOLD=""
    COLOR_RESET=""
fi

log_info()  { echo "${COLOR_CYAN}[INFO]${COLOR_RESET} $*"; }
log_warn()  { echo "${COLOR_YELLOW}[WARN]${COLOR_RESET} $*"; }
log_error() { echo "${COLOR_RED}[ERROR]${COLOR_RESET} $*" >&2; }
log_ok()    { echo "${COLOR_GREEN}[OK]${COLOR_RESET} $*"; }

# 脚本所在目录，避免依赖当前工作目录
SCRIPT_PATH="$(cd "$(dirname "$0")" && pwd)"
SERVER_PATH="$SCRIPT_PATH"
SERVER_NAME="$(basename "$SERVER_PATH")"

SERVER_BIN_PATH="${SERVER_PATH}/"
# 可通过环境变量 SERVER_CONF 覆盖默认配置文件路径
SERVER_PARAM="${SERVER_CONF:-/data/GoOne/commconf/server_conf.yaml}"
SERVER_PARAM_OTHER="${SERVER_PATH}/${SERVER_NAME}_conf2.json"

echo "${COLOR_BOLD}${COLOR_BLUE}========== GoOne Server Control ==========${COLOR_RESET}"
log_info "Server name   : ${SERVER_NAME}"
log_info "Server bin dir: ${SERVER_BIN_PATH}"
log_info "Server conf   : ${SERVER_PARAM}"

is_running()
{
    # 优先使用 pid 文件，如果存在的话
    if [ -f "${SERVER_NAME}.pid" ]; then
        pid=$(cat "${SERVER_NAME}.pid" 2>/dev/null || echo "")
        if [ -n "$pid" ] && ps -p "$pid" -o comm= 2>/dev/null | grep -qw "$SERVER_NAME"; then
            log_ok "Server ${SERVER_NAME} is running with pid ${pid}"
            return 0
        fi
    fi

    # 回退到进程名检测
    proc_num=$(ps -C "${SERVER_NAME}" 2>/dev/null | sed -e '1d' | wc -l)
    if [ "${proc_num}" -gt 0 ]; then
        log_ok "Server ${SERVER_NAME} is already running!"
        return 0
    else
        return 1
    fi
}

start()
{
    is_running
    if [ $? -eq 1 ]; then
        if ! command -v daemonize >/dev/null 2>&1; then
            log_error "Start server ${SERVER_NAME} FAILED: 'daemonize' not found in PATH"
            return 1
        fi

        daemonize -e ./err.log -c ./ "${SERVER_BIN_PATH}${SERVER_NAME}" -svr_conf="${SERVER_PARAM}"
        if [ $? -eq 0 ];then
            ps -C "$SERVER_NAME" -o "pid=" > ${SERVER_NAME}.pid
            log_ok "Start server ${SERVER_NAME} OK"
        else
            log_error "Start server ${SERVER_NAME} FAILED"
        fi
    else
        log_warn "Start server ${SERVER_NAME} FAILED: already running"
    fi
}

start2()
{
    is_running
    if [ $? -eq 1 ]; then
        if ! command -v daemonize >/dev/null 2>&1; then
            log_error "Start server ${SERVER_NAME} FAILED: 'daemonize' not found in PATH"
            return 1
        fi

        daemonize -e ./err.log -c ./ "${SERVER_BIN_PATH}${SERVER_NAME}" -svr_conf="${SERVER_PARAM}" -pay_conf="${SERVER_PARAM_OTHER}"
        if [ $? -eq 0 ];then
            ps -C "$SERVER_NAME" -o "pid=" > ${SERVER_NAME}.pid
            log_ok "Start server ${SERVER_NAME} OK (with pay_conf=${SERVER_PARAM_OTHER})"
        else
            log_error "Start server ${SERVER_NAME} FAILED"
        fi
    else
        log_warn "Start server ${SERVER_NAME} FAILED: already running"
    fi
}


stop()
{
    i=3
    stop_flag=0
    while [ $i -gt 0 ]
    do
        killall "${SERVER_NAME}" >/dev/null 2>&1 || true
        sleep 1

        is_running
        if [ $? -eq 1 ]; then
            stop_flag=1
            break
        fi

        ((i=i-1))
    done
    if [ ${stop_flag} -eq 0 ] ; then
        killall -9 "${SERVER_NAME}" >/dev/null 2>&1 || true
        is_running
        # is_running 返回 1 表示已经不在运行
        if [ $? -eq 1 ]; then
            stop_flag=1
        fi

    fi

    if [ $stop_flag -eq 1 ];then
        rm -f "${SERVER_NAME}.pid"
        log_ok "Stop server ${SERVER_NAME} OK"
    else
        log_error "Stop server ${SERVER_NAME} FAILED"
    fi

    return 0
}

#clean()
#{
#    str=`grep key ${SERVER_PARAM} | grep shm | awk -F':' '{print $2}'`
#    for key in $str; do
#        ipcrm -M $key
#    done
#}

reload()
{
    is_running
    if [ $? -eq 0 ]; then
        log_warn "Server ${SERVER_NAME} is not running"
    else
        if [ -f "${SERVER_NAME}.pid" ]; then
            pid=$(cat "${SERVER_NAME}.pid" 2>/dev/null || echo "")
            if [ -n "$pid" ]; then
                kill -s SIGUSR1 "$pid"
            else
                kill -s SIGUSR1 "${SERVER_NAME}"
            fi
        else
            kill -s SIGUSR1 "${SERVER_NAME}"
        fi
    fi
}

usage()
{
    echo "Usage: $0 [start|stop|restart|check]"
}

if [ $# -lt 1 ];then
    usage
    exit
fi

if [ "$1" = "start" ];then
    start
elif [ "$1" = "start2" ];then
    start2
elif [ "$1" = "stop" ];then
    stop
elif [ "$1" = "restart" ];then
    stop
    start
elif [ "$1" = "restart2" ];then
    stop
    start2
elif [ "$1" = "check" ];then
    is_running
    exit $?
elif [ "$1" = "reload" ];then
    reload
else
    usage
fi
