# Future Features

记录值得未来实现的功能点，按优先级排序。

## P1: 自动 Stage + Execute

**来源**: `cmd/args.go:43-51` 中保留的 FUTURE 注释 (`registerStageFlags`)

**意图**: 通过 SMB 上传本地文件到远程主机，然后自动执行。一步完成"上传+执行"流程。

**价值**:
- 当前用户需要手动上传文件再执行，体验割裂
- 竞品（impacket psexec/smbexec）都内置此能力
- 是远程执行工具的核心功能之一

**前置依赖**:
- `pkg/goexec/smb/` 当前只有读取能力（用于输出抓取），需要先实现 SMB 写入支持
- 需要 `go-smb2.fork` 库的写入 API

**设计思路**:
1. `smb.Client` 新增 `WriteFile(share, remotePath string, data []byte)` 方法
2. `cmd/args.go` 新增 `registerStageFlags`，提供 `--stage` / `-E` flag 指定本地文件
3. 在 `PersistentPreRunE` 中，如果指定了 `--stage`，自动上传到 `ADMIN$\Temp\<random>` 并将 `exec.Input.Executable` 设为远程路径
4. 执行完成后自动清理远程文件（通过 Cleaner）

**工作量**: 中等（SMB 写入 + CLI 集成 + 清理逻辑）

---

## P2: Actions 自定义 XML 序列化

**来源**: 已删除的 `pkg/goexec/tsch/task/action.go` 中 `MarshalXML`/`UnmarshalXML` 方法

**意图**: 为 `Actions` 类型提供自定义 XML 序列化，保证子元素输出顺序、正确处理未知元素。

**何时需要**:
- 当遇到真实 Windows 服务器返回的复杂任务 XML（含 ComHandler、SendEmail、ShowMessage 等混合 action 类型，或含未知扩展元素）
- 当 `tsch change` 命令在修改复杂任务时丢失原有 action 信息

**当前状态**: 标准库 `encoding/xml` 通过结构体标签能正确处理已知的四种 action 类型。尚未有真实场景报告问题。

**触发条件**: 出现用户报告 `tsch change` 修改任务后原有非 Exec 类型的 action 丢失时再实现。

**工作量**: 小（代码已有历史参考，约 110 行）
