package table

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"
)

func TestAllModelsRegistered(t *testing.T) {
	// 1. 获取所有已注册的模型名称
	registered := make(map[string]bool)
	for _, m := range GetAllModels() {
		name := reflect.TypeOf(m).Elem().Name()
		registered[name] = true
	}

	// 2. 解析当前目录下的所有 Go 文件
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, 0)
	if err != nil {
		t.Fatalf("Failed to parse package: %v", err)
	}

	// 3. 遍历代码结构，查找未注册的模型
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				// 查找结构体定义
				typeSpec, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					return true
				}

				// 检查是否嵌入了 gorm.Model
				if hasGormModel(structType) {
					modelName := typeSpec.Name.Name
					if !registered[modelName] {
						t.Errorf("Model '%s' is defined but not registered in GetAllModels()", modelName)
					}
				}
				return true
			})
		}
	}
}

// hasGormModel 检查结构体是否包含 gorm.Model 匿名字段
func hasGormModel(structType *ast.StructType) bool {
	for _, field := range structType.Fields.List {
		// 匿名字段没有名称
		if len(field.Names) == 0 {
			if selExpr, ok := field.Type.(*ast.SelectorExpr); ok {
				if x, ok := selExpr.X.(*ast.Ident); ok {
					if x.Name == "gorm" && selExpr.Sel.Name == "Model" {
						return true
					}
				}
			}
		}
	}
	return false
}
