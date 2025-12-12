## GoOne 部署说明（deploy 目录）

### 1. 目录结构

- **deploy.sh**：主部署脚本，封装 Ansible，对各环境 / 各服务做 `init|push|start|stop|restart`。
- **playbook_dev/**：不同环境的 Ansible playbook 与变量（如 `dev1.yml`、`dev1_vars`）。
- **roles/**：各服务对应的 Ansible role（`mainsvr`、`websvr` 等）。
- **hosts/**：Ansible inventory 文件（如 `host_dev.txt`）。
- **inithost/**：初始化操作系统的 Ansible playbook（如基础依赖、limit、时区等）。
- **scripts/server.sh**：部署到目标机上后，用于单个服务的启动 / 停止 / 检查 / reload。

### 2. deploy.sh 用法

```bash
./deploy.sh <env> <init|push|start|stop|restart> [role names...]
```

- **示例**
  - 初始化所有服务（dev1 环境）  
    `./deploy.sh dev1 init`
  - 只部署并重启 `mainsvr`、`connsvr`  
    `./deploy.sh dev1 restart mainsvr connsvr`

- **说明**
  - `env`：对应 `playbook_dev/<env>.yml`，比如 `dev1`、`dev2`、`dev_local`。
  - 不指定 role 时，会对 `ALL_TARGET_ROLES` 中列出的所有服务生效。
  - 如果存在 `hosts/host_<env>.txt` 会优先用该文件，否则使用 `hosts/host_dev.txt`。
  - 输出中会高亮显示：
    - Env / Option / Target roles / Hosts file / Ansible tags
  - 设置 `NO_COLOR=1` 可关闭所有彩色输出。

### 3. scripts/server.sh 用法（单机服务管理）

在目标服务器上，每个服务目录下通常有一个 `server.sh`，例如：

```bash
cd /data/GoOne/bin/mainsvr
./server.sh start      # 启动
./server.sh stop       # 停止
./server.sh restart    # 重启
./server.sh check      # 检查运行状态
./server.sh reload     # 发 SIGUSR1 做热加载（如果服务支持）
```

- **配置路径**
  - 默认配置文件：`/data/GoOne/commconf/server_conf.yaml`
  - 可通过环境变量覆盖：  
    `export SERVER_CONF=/data/GoOne/commconf/server_conf_dev1.yaml`
  - `start2/restart2` 会额外带上 `-pay_conf=${SERVER_NAME}_conf2.json`

- **输出风格**
  - 统一使用：`[INFO]` / `[OK]` / `[WARN]` / `[ERROR]` 前缀，并带颜色。
  - 依赖 `daemonize` 命令启动进程，如果未安装会有明确错误提示。

### 4. inithost 目录

- `inithost/host.txt`：初始化阶段使用的主机列表。
- `inithost/inithost.yml`：执行基础初始化的 playbook。

可以在部署机上单独运行初始化（示例命令视你当前 Ansible 版本和路径略有不同，可以根据你们现有习惯调整）：

```bash
cd deploy/inithost
ansible-playbook -i host.txt inithost.yml
```

### 5. 常见问题

- **提示某个 role 未在已知列表中**
  - `deploy.sh` 会输出 `[WARN] role 'xxx' is not in known role list`，可能是：
    - `roles/` 下没有对应目录；
    - role 名拼写错误。
- **ansible-playbook 失败**
  - 终端会显示红色 `[ERROR] ansible-playbook failed.`，可以上滚查看具体失败 task。


