package teaconfigs

import (
	"errors"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/files"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/utils/string"
	"math/rand"
	"strings"
	"time"
)

// 服务配置
type ServerConfig struct {
	On bool `yaml:"on" json:"on"` // 是否开启 @TODO

	Id          string   `yaml:"id" json:"id"`                   // ID
	Description string   `yaml:"description" json:"description"` // 描述
	Name        []string `yaml:"name" json:"name"`               // 域名
	Http        bool     `yaml:"http" json:"http"`               // 是否支持HTTP

	// 监听地址
	// @TODO 支持参数，比如：127.0.01:1234?ssl=off
	Listen []string `yaml:"listen" json:"listen"`

	Root      string                 `yaml:"root" json:"root"`           // 资源根目录 @TODO
	Index     []string               `yaml:"index" json:"index"`         // 默认文件 @TODO
	Charset   string                 `yaml:"charset" json:"charset"`     // 字符集 @TODO
	Backends  []*ServerBackendConfig `yaml:"backends" json:"backends"`   // 后端服务器配置
	Locations []*LocationConfig      `yaml:"locations" json:"locations"` // 地址配置

	Async   bool     `yaml:"async" json:"async"`     // 请求是否异步处理 @TODO
	Notify  []string `yaml:"notify" json:"notify"`   // 请求转发地址 @TODO
	LogOnly bool     `yaml:"logOnly" json:"logOnly"` // 是否只记录日志 @TODO

	// 访问日志
	AccessLog []*AccessLogConfig `yaml:"accessLog" json:"accessLog"` // 访问日志

	// @TODO 支持ErrorLog

	// SSL
	SSL *SSLConfig `yaml:"ssl" json:"ssl"`

	Headers       []*HeaderConfig `yaml:"headers" json:"headers"`             // 自定义Header
	IgnoreHeaders []string        `yaml:"ignoreHeaders" json:"ignoreHeaders"` // 忽略的Header TODO

	// 参考：http://nginx.org/en/docs/http/ngx_http_access_module.html
	Allow []string `yaml:"allow" json:"allow"` //TODO
	Deny  []string `yaml:"deny" json:"deny"`   //TODO

	Filename string `yaml:"filename" json:"filename"` // 配置文件名

	Rewrite []*RewriteRule   `yaml:"rewrite" json:"rewrite"` // 重写规则 TODO
	Fastcgi []*FastcgiConfig `yaml:"fastcgi" json:"fastcgi"` // Fastcgi配置 TODO
	Proxy   string           `yaml:"proxy" json:"proxy"`     //  代理配置 TODO

	// API相关
	APIOn         bool      `yaml:"apiOn" json:"apiOn"`               // 是否开启API功能
	APIFiles      []string  `yaml:"apiFiles" json:"apiFiles"`         // API文件列表
	APIGroups     []string  `yaml:"apiGroups" json:"apiGroups"`       // API分组
	APIVersions   []string  `yaml:"apiVersions" json:"apiVersions"`   // API版本
	APITestPlans  []string  `yaml:"apiTestPlans" json:"apiTestPlans"` // API测试计划
	APILimit      *APILimit `yaml:"apiLimit" json:"apiLimit"`         // API全局的限制 TODO
	apiPathMap    map[string]*API                                     // path => api
	apiPatternMap map[string]*API                                     // path => api
}

// 从目录中加载配置
func LoadServerConfigsFromDir(dirPath string) []*ServerConfig {
	servers := []*ServerConfig{}

	dir := files.NewFile(dirPath)
	subFiles := dir.Glob("*.proxy.conf")
	files.Sort(subFiles, files.SortTypeModifiedTimeReverse)
	for _, configFile := range subFiles {
		reader, err := configFile.Reader()
		if err != nil {
			logs.Error(err)
			continue
		}

		config := &ServerConfig{}
		err = reader.ReadYAML(config)
		if err != nil {
			continue
		}
		config.Filename = configFile.Name()
		servers = append(servers, config)
	}

	return servers
}

// 取得一个新的服务配置
func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		On: true,
		Id: stringutil.Rand(16),
	}
}

// 从配置文件中读取配置信息
func NewServerConfigFromFile(filename string) (*ServerConfig, error) {
	if len(filename) == 0 {
		return nil, errors.New("filename should not be empty")
	}
	reader, err := files.NewReader(Tea.ConfigFile(filename))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	config := &ServerConfig{}
	err = reader.ReadYAML(config)
	if err != nil {
		return nil, err
	}

	config.Filename = filename

	// 初始化
	if len(config.Locations) == 0 {
		config.Locations = []*LocationConfig{}
	}
	if len(config.Headers) == 0 {
		config.Headers = []*HeaderConfig{}
	}
	if len(config.IgnoreHeaders) == 0 {
		config.IgnoreHeaders = []string{}
	}

	return config, nil
}

// 校验配置
func (this *ServerConfig) Validate() error {
	// ssl
	if this.SSL != nil {
		err := this.SSL.Validate()
		if err != nil {
			return err
		}
	}

	// backends
	for _, backend := range this.Backends {
		err := backend.Validate()
		if err != nil {
			return err
		}
	}

	// locations
	for _, location := range this.Locations {
		err := location.Validate()
		if err != nil {
			return err
		}
	}

	// fastcgi
	for _, fastcgi := range this.Fastcgi {
		err := fastcgi.Validate()
		if err != nil {
			return err
		}
	}

	// rewrite rules
	for _, rewriteRule := range this.Rewrite {
		err := rewriteRule.Validate()
		if err != nil {
			return err
		}
	}

	// headers
	for _, header := range this.Headers {
		err := header.Validate()
		if err != nil {
			return err
		}
	}

	// api
	this.apiPathMap = map[string]*API{}
	this.apiPatternMap = map[string]*API{}
	for _, apiFilename := range this.APIFiles {
		api := NewAPIFromFile(apiFilename)
		if api == nil {
			continue
		}
		err := api.Validate()
		if err != nil {
			return err
		}
		if api.pathReg == nil {
			this.apiPathMap[api.Path] = api
		} else {
			this.apiPatternMap[api.Path] = api
		}
	}

	// api limit
	if this.APILimit != nil {
		err := this.APILimit.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

// 添加域名
func (this *ServerConfig) AddName(name ... string) {
	this.Name = append(this.Name, name ...)
}

// 添加监听地址
func (this *ServerConfig) AddListen(address string) {
	this.Listen = append(this.Listen, address)
}

// 添加后端服务
func (this *ServerConfig) AddBackend(config *ServerBackendConfig) {
	this.Backends = append(this.Backends, config)
}

// 取得下一个可用的后端服务
// @TODO 实现backend中的各种参数
func (this *ServerConfig) NextBackend() *ServerBackendConfig {
	if len(this.Backends) == 0 {
		return nil
	}

	availableBackends := []*ServerBackendConfig{}
	for _, backend := range this.Backends {
		if backend.On && !backend.IsDown {
			availableBackends = append(availableBackends, backend)
		}
	}

	countBackends := len(availableBackends)
	if countBackends == 0 {
		return nil
	}

	rand.Seed(time.Now().UnixNano())
	index := rand.Int() % countBackends
	return availableBackends[index]
}

// 设置Header
func (this *ServerConfig) SetHeader(name string, value string) {
	found := false
	upperName := strings.ToUpper(name)
	for _, header := range this.Headers {
		if strings.ToUpper(header.Name) == upperName {
			found = true
			header.Value = value
		}
	}
	if found {
		return
	}

	header := NewHeaderConfig()
	header.Name = name
	header.Value = value
	this.Headers = append(this.Headers, header)
}

// 删除指定位置上的Header
func (this *ServerConfig) DeleteHeaderAtIndex(index int) {
	if index >= 0 && index < len(this.Headers) {
		this.Headers = lists.Remove(this.Headers, index).([]*HeaderConfig)
	}
}

// 取得指定位置上的Header
func (this *ServerConfig) HeaderAtIndex(index int) *HeaderConfig {
	if index >= 0 && index < len(this.Headers) {
		return this.Headers[index]
	}
	return nil
}

// 格式化Header
func (this *ServerConfig) FormatHeaders(formatter func(source string) string) []*HeaderConfig {
	result := []*HeaderConfig{}
	for _, header := range this.Headers {
		result = append(result, &HeaderConfig{
			Name:   header.Name,
			Value:  formatter(header.Value),
			Always: header.Always,
			Status: header.Status,
		})
	}
	return result
}

// 添加一个自定义Header
func (this *ServerConfig) AddHeader(header *HeaderConfig) {
	this.Headers = append(this.Headers, header)
}

// 屏蔽一个Header
func (this *ServerConfig) AddIgnoreHeader(name string) {
	this.IgnoreHeaders = append(this.IgnoreHeaders, name)
}

// 移除对Header的屏蔽
func (this *ServerConfig) DeleteIgnoreHeaderAtIndex(index int) {
	if index >= 0 && index < len(this.IgnoreHeaders) {
		this.IgnoreHeaders = lists.Remove(this.IgnoreHeaders, index).([]string)
	}
}

// 更改Header的屏蔽
func (this *ServerConfig) UpdateIgnoreHeaderAtIndex(index int, name string) {
	if index >= 0 && index < len(this.IgnoreHeaders) {
		this.IgnoreHeaders[index] = name
	}
}

// 获取某个位置上的配置
func (this *ServerConfig) LocationAtIndex(index int) *LocationConfig {
	if index < 0 {
		return nil
	}
	if index >= len(this.Locations) {
		return nil
	}
	location := this.Locations[index]
	location.Validate()
	return location
}

// 将配置写入文件
func (this *ServerConfig) WriteToFile(path string) error {
	writer, err := files.NewWriter(path)
	if err != nil {
		return err
	}
	_, err = writer.WriteYAML(this)
	writer.Close()
	return err
}

// 将配置写入文件
func (this *ServerConfig) WriteToFilename(filename string) error {
	writer, err := files.NewWriter(Tea.ConfigFile(filename))
	if err != nil {
		return err
	}
	_, err = writer.WriteYAML(this)
	writer.Close()
	return err
}

// 保存
func (this *ServerConfig) Save() error {
	if len(this.Filename) == 0 {
		return errors.New("'filename' should be specified")
	}

	return this.WriteToFilename(this.Filename)
}

// 判断是否和域名匹配
// @TODO 支持  .example.com （所有以example.com结尾的域名，包括example.com）
// 更多参考：http://nginx.org/en/docs/http/ngx_http_core_module.html#server_name
func (this *ServerConfig) MatchName(name string) (matchedName string, matched bool) {
	if len(name) == 0 {
		return "", false
	}
	pieces1 := strings.Split(name, ".")
	countPieces1 := len(pieces1)
	for _, testName := range this.Name {
		if len(testName) == 0 {
			continue
		}
		if name == testName {
			return testName, true
		}
		pieces2 := strings.Split(testName, ".")
		if countPieces1 != len(pieces2) {
			continue
		}
		matched := true
		for index, piece := range pieces2 {
			if pieces1[index] != piece && piece != "*" && piece != "" {
				matched = false
				break
			}
		}
		if matched {
			return "", true
		}
	}
	return "", false
}

// 取得第一个非泛解析的域名
func (this *ServerConfig) FirstName() string {
	for _, name := range this.Name {
		if strings.Contains(name, "*") {
			continue
		}
		return name
	}
	return ""
}

// 取得下一个可用的fastcgi
// @TODO 实现fastcgi中的各种参数
func (this *ServerConfig) NextFastcgi() *FastcgiConfig {
	countFastcgi := len(this.Fastcgi)
	if countFastcgi == 0 {
		return nil
	}
	rand.Seed(time.Now().UnixNano())
	index := rand.Int() % countFastcgi
	return this.Fastcgi[index]
}

// 添加路径规则
func (this *ServerConfig) AddLocation(location *LocationConfig) {
	this.Locations = append(this.Locations, location)
}

// 添加API
func (this *ServerConfig) AddAPI(api *API) {
	if api == nil {
		return
	}

	// 分析API
	if this.apiPathMap != nil {
		err := api.Validate()
		if err == nil {
			if api.pathReg == nil {
				this.apiPathMap[api.Path] = api
			} else {
				this.apiPatternMap[api.Path] = api
			}
		}
	}

	// 如果已包含文件名则不重复添加
	if lists.Contains(this.APIFiles, api.Filename) {
		return
	}
	this.APIFiles = append(this.APIFiles, api.Filename)
}

// 获取所有APIs
func (this *ServerConfig) FindAllAPIs() []*API {
	apis := []*API{}
	for _, filename := range this.APIFiles {
		api := NewAPIFromFile(filename)
		if api == nil {
			continue
		}
		apis = append(apis, api)
	}
	return apis
}

// 获取单个API信息
func (this *ServerConfig) FindAPI(path string) *API {
	for _, api := range this.FindAllAPIs() {
		if api.Path == path {
			return api
		}
	}
	return nil
}

// 查找激活状态中的API
func (this *ServerConfig) FindActiveAPI(path string, method string) (api *API, params map[string]string) {
	api, found := this.apiPathMap[path]
	if !found {
		// 寻找pattern
		for _, api := range this.apiPatternMap {
			params, found := api.Match(path)
			if !found || api.IsDeprecated || !api.On || !api.AllowMethod(method) {
				continue
			}
			return api, params
		}

		return nil, nil
	}

	// 检查是否过期或者失效
	if api.IsDeprecated || !api.On || !api.AllowMethod(method) {
		return nil, nil
	}

	return api, nil
}

// 删除API
func (this *ServerConfig) DeleteAPI(api *API) {
	this.APIFiles = lists.Delete(this.APIFiles, api.Filename).([]string)

	delete(this.apiPathMap, api.Path)
	delete(this.apiPatternMap, api.Path)
}

// 添加API分组
func (this *ServerConfig) AddAPIGroup(name string) {
	this.APIGroups = append(this.APIGroups, name)
}

// 删除API分组
func (this *ServerConfig) RemoveAPIGroup(name string) {
	result := []string{}
	for _, groupName := range this.APIGroups {
		if groupName != name {
			result = append(result, groupName)
		}
	}

	for _, filename := range this.APIFiles {
		api := NewAPIFromFile(filename)
		if api == nil {
			continue
		}
		api.RemoveGroup(name)
		api.Save()
	}

	this.APIGroups = result
}

// 修改API分组
func (this *ServerConfig) ChangeAPIGroup(oldName string, newName string) {
	result := []string{}
	for _, groupName := range this.APIGroups {
		if groupName == oldName {
			result = append(result, newName)
		} else {
			result = append(result, groupName)
		}
	}

	for _, filename := range this.APIFiles {
		api := NewAPIFromFile(filename)
		if api == nil {
			continue
		}
		api.ChangeGroup(oldName, newName)
		api.Save()
	}

	this.APIGroups = result
}

// 把API分组往上调整
func (this *ServerConfig) MoveUpAPIGroup(name string) {
	index := lists.Index(this.APIGroups, name)
	if index <= 0 {
		return
	}
	this.APIGroups[index], this.APIGroups[index-1] = this.APIGroups[index-1], this.APIGroups[index]
}

// 把API分组往下调整
func (this *ServerConfig) MoveDownAPIGroup(name string) {
	index := lists.Index(this.APIGroups, name)
	if index < 0 {
		return
	}
	this.APIGroups[index], this.APIGroups[index+1] = this.APIGroups[index+1], this.APIGroups[index]
}

// 添加API版本
func (this *ServerConfig) AddAPIVersion(name string) {
	this.APIVersions = append(this.APIVersions, name)
}

// 删除API版本
func (this *ServerConfig) RemoveAPIVersion(name string) {
	result := []string{}
	for _, versionName := range this.APIVersions {
		if versionName != name {
			result = append(result, versionName)
		}
	}

	for _, filename := range this.APIFiles {
		api := NewAPIFromFile(filename)
		if api == nil {
			continue
		}
		api.RemoveVersion(name)
		api.Save()
	}

	this.APIVersions = result
}

// 修改API版本
func (this *ServerConfig) ChangeAPIVersion(oldName string, newName string) {
	result := []string{}
	for _, versionName := range this.APIVersions {
		if versionName == oldName {
			result = append(result, newName)
		} else {
			result = append(result, versionName)
		}
	}

	for _, filename := range this.APIFiles {
		api := NewAPIFromFile(filename)
		if api == nil {
			continue
		}
		api.ChangeVersion(oldName, newName)
		api.Save()
	}

	this.APIVersions = result
}

// 把API版本往上调整
func (this *ServerConfig) MoveUpAPIVersion(name string) {
	index := lists.Index(this.APIVersions, name)
	if index <= 0 {
		return
	}
	this.APIVersions[index], this.APIVersions[index-1] = this.APIVersions[index-1], this.APIVersions[index]
}

// 把API版本往下调整
func (this *ServerConfig) MoveDownAPIVersion(name string) {
	index := lists.Index(this.APIVersions, name)
	if index < 0 {
		return
	}
	this.APIVersions[index], this.APIVersions[index+1] = this.APIVersions[index+1], this.APIVersions[index]
}

// 添加测试计划
func (this *ServerConfig) AddTestPlan(filename string) {
	this.APITestPlans = append(this.APITestPlans, filename)
}

// 查找所有测试计划
func (this *ServerConfig) FindTestPlans() []*APITestPlan {
	result := []*APITestPlan{}
	for _, filename := range this.APITestPlans {
		plan := NewAPITestPlanFromFile(filename)
		if plan != nil {
			result = append(result, plan)
		}
	}
	return result
}

// 删除某个测试计划
func (this *ServerConfig) DeleteTestPlan(filename string) error {
	if len(filename) == 0 {
		return errors.New("filename should not be empty")
	}

	plan := NewAPITestPlanFromFile(filename)
	if plan != nil {
		err := plan.Delete()
		if err != nil {
			return err
		}
	}

	this.APITestPlans = lists.Delete(this.APITestPlans, filename).([]string)

	return nil
}
