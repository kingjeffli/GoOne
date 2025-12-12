#!/bin/bash
#set -x

set -euo pipefail
export ANSIBLE_HOST_KEY_CHECKING=False

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

ENV=${1:-}
OPTION=${2:-}

usage()
{
  echo "Usage: $0 <env> <init|push|start|stop|restart> [role names]..."
  echo "Example: $0 dev1 push mainsvr connsvr  // push file of mainsvr and connsvr"
  echo "         $0 dev1 init  // init all server, no role indicated means all roles"
}

if [[ $# -lt 2 ]]; then
  usage
  exit 1
fi

# 所有的 role（请保持与 roles/ 目录同步）
ALL_TARGET_ROLES=("commconf" "gamedata" "connsvr" "mainsvr" "infosvr" "mysqlsvr" "gamesvr" "mailsvr" "friendsvr" "chatsvr" "websvr" "guildsvr" "roomcentersvr" "texassvr")

VALID_OPTIONS=("init" "push" "start" "stop" "restart")

# 校验 OPTION 是否合法，防止 tag 拼错
is_valid_option=false
for op in "${VALID_OPTIONS[@]}"; do
  if [[ "$OPTION" == "$op" ]]; then
    is_valid_option=true
    break
  fi
done

if [[ "$is_valid_option" != true ]]; then
  echo "${COLOR_RED}[ERROR] invalid option '$OPTION'. Supported: ${VALID_OPTIONS[*]}${COLOR_RESET}" >&2
  usage
  exit 1
fi

#如果命令行没有指明role，则默认为所有role
target_role=()
if [[ $# -lt 3 ]]; then
    # 全量 role
    target_role=("${ALL_TARGET_ROLES[@]}")
else
    # 从命令行获取目标 role
    for ((i=3;i<=$#;i++));
    do
       target_role[${#target_role[@]}]=${!i}
    done
fi

# 检查是否有非法 role 名称，避免 ansible tag 拼写错误导致悄悄不执行
for r in "${target_role[@]}"; do
  found=false
  for ar in "${ALL_TARGET_ROLES[@]}"; do
    if [[ "$r" == "$ar" ]]; then
      found=true
      break
    fi
  done
  if [[ "$found" != true ]]; then
    echo "${COLOR_YELLOW}[WARN] role '$r' is not in known role list, please confirm it is correct.${COLOR_RESET}" >&2
  fi
done

#计算tags
target=""
for i in "${!target_role[@]}";  
do
    #最前面不用添加逗号
    if [[ $i != 0 ]]; then
        target="$target,"
    fi
    target="$target${target_role[$i]}_$OPTION"
done

#如果没有tags，则在ansible后面不添加--tags标签
tags="--tags $target"
if [[ $target == "" ]]; then
  tags=""
fi

# 目录与文件检查
PLAYBOOK_DIR="playbook_dev"
DEFAULT_HOSTS_FILE="hosts/host_dev.txt"
ENV_HOSTS_FILE="hosts/host_${ENV}.txt"

PLAYBOOK_FILE="${PLAYBOOK_DIR}/${ENV}.yml"

if [[ ! -f "$PLAYBOOK_FILE" ]]; then
  echo "${COLOR_RED}[ERROR] playbook file '$PLAYBOOK_FILE' not found.${COLOR_RESET}" >&2
  exit 1
fi

if [[ -f "$ENV_HOSTS_FILE" ]]; then
  HOSTS_FILE="$ENV_HOSTS_FILE"
else
  HOSTS_FILE="$DEFAULT_HOSTS_FILE"
fi

if [[ ! -f "$HOSTS_FILE" ]]; then
  echo "${COLOR_RED}[ERROR] hosts file '$HOSTS_FILE' not found.${COLOR_RESET}" >&2
  exit 1
fi

echo "${COLOR_BOLD}${COLOR_BLUE}========== GoOne Ansible Deploy ==========${COLOR_RESET}"
echo "${COLOR_BOLD} Env:           ${COLOR_YELLOW}${ENV}${COLOR_RESET}"
echo "${COLOR_BOLD} Option:        ${COLOR_CYAN}${OPTION}${COLOR_RESET}"
echo "${COLOR_BOLD} Target roles:  ${COLOR_GREEN}${target_role[*]}${COLOR_RESET}"
echo "${COLOR_BOLD} Hosts file:    ${COLOR_CYAN}${HOSTS_FILE}${COLOR_RESET}"
if [[ -n "$tags" ]]; then
  echo "${COLOR_BOLD} Ansible tags:  ${COLOR_GREEN}${target}${COLOR_RESET}"
else
  echo "${COLOR_BOLD} Ansible tags:  ${COLOR_YELLOW}<none>${COLOR_RESET} (all tasks in playbook)"
fi
echo "${COLOR_BLUE}------------------------------------------${COLOR_RESET}"

# 临时文件清理函数
TMP=""
cleanup() {
  if [[ -n "${TMP:-}" && -f "$TMP" ]]; then
    rm -f "$TMP"
  fi
}
trap cleanup EXIT

#由于在子目录运行playbook会有目录结构问题，所以建个临时文件
TMP="$(mktemp .tmpXXXXXX.myl)"
cp "$PLAYBOOK_FILE" "$TMP"

#执行playbook
if ! ansible-playbook -i "$HOSTS_FILE" "$TMP" $tags; then
  echo "${COLOR_RED}[ERROR] ansible-playbook failed.${COLOR_RESET}" >&2
  exit 1
fi

echo "${COLOR_GREEN}[OK] Deploy completed for env=${ENV}, option=${OPTION}.${COLOR_RESET}"


