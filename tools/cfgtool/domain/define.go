package domain

const (
	// 类类型
	TypeOfConfig = 4
	TypeOfStruct = 3
	TypeOfEnum   = 2
	TypeOfBase   = 1

	// 值类型
	ValueOfBase  = 1
	ValueOfList  = 2
	ValueOfMap   = 3
	ValueOfGroup = 4

	MaxArrayDepth = 3
)

var (
	Version      = "1.0.6"       // 当前版本号
	Module       = ""            // 项目目录
	ConfMode     = ""            // 配置gen模式（all：全部  client：客户端  server：服务器）；xlsx第四行支持 key/KEY 标记主键索引
	ProtoPkgName = "g1.protocol" // proto包名
	PkgName      = ""            // 包名
	XlsxPath     = ""            // 解析文件路径
	ProtoPath    = ""            // proto文件路径
	PbPath       = ""            // proto生成路径
	CodePath     = ""            // 代码生成路径
	CppPath      = ""            // C++代码生成路径
	NodeJsPath   = ""            // Node.js/TypeScript代码生成路径
	JsonPath     = ""            // 数据文件路径
	BytesPath    = ""            // 数据文件路径
	TextPath     = ""            // 数据文件路径
	LuaPath      = ""            // lua数据文件路径
	TsPath       = ""            // ts数据文件路径
)
