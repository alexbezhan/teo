package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
	// "unicode"
)

func main() {
    args := os.Args[1:]

    if len(args) == 0 {
    	fmt.Println("Teo is a tool for managing Teo source code.")
        return
    }

    dir := args[0]

    filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
        if !strings.Contains(entry.Name(), ".teo") {
            return nil
        }

        fmt.Println("Read   ", path)

        var pathWithoutExtension string
        {
            suffix := ".teo"
            path, found := strings.CutSuffix(path, suffix)
            if !found {
                panic(fmt.Errorf("should found %v", suffix))
            }
            pathWithoutExtension = path
        }
        filePathOut := pathWithoutExtension + ".go"
        fileBytes, err := os.ReadFile(path)
        if err != nil {
            fmt.Println(err)
        }
        fileText := string(fileBytes)
        compileSourceFile(fileText, filePathOut)

        return nil
    })
}

func compileSourceFile(fileText string, filePathOut string) {
    substrings := strings.Fields(fileText)
    var tokens []Token
    for _, substr := range substrings {
        tokens = append(tokens, Token{tpe: TokenTypeNone, text: substr})
    }

    // identify tokens and append
    var tokensNew []Token
    for i := range tokens {
        var tokenPrev Token
        if i > 0 {
            tokenPrev = tokens[i-1]
        } else {
            tokenPrev = tokenNew()
        }

        token0 := &tokens[i]

        var token1 *Token
        if i < len(tokens) -1 {
            token1 = &tokens[i+1]
        } else {
            tkn := tokenNew()
            token1 = &tkn
        }

        // fmt.Println(token0)

        if token0.tpe == TokenTypeName {
            tokensNew = append(tokensNew, *token0)
        } else if token0.tpe == TokenTypeStringLiteral {
            tokensNew = append(tokensNew, *token0)
        } else if token0.text == "package" {
            token0.tpe = TokenTypePackage
            token1.tpe = TokenTypeName

            tokensNew = append(tokensNew, *token0)
        } else if token0.text == "import" {
            token0.tpe = TokenTypeImport
            token1.tpe = TokenTypeStringLiteral

            tokensNew = append(tokensNew, *token0)
        } else if token0.text == "function" {
            token0.tpe = TokenTypeFunction

            token1Name, token1Args, parenFound := strings.Cut(token1.text, "(")
            if parenFound == false {
                panic("function is missing arguments ()")
            }

            tokensNew = append(tokensNew, *token0)
            tokensNew = append(tokensNew, Token{tpe: TokenTypeName, text: token1Name})
            tokensNew = append(tokensNew, Token{tpe: TokenTypeArgs, text: "(" + token1Args})
        } else if token0.text == "{" {
            token0.tpe = TokenTypeScopeOpen

            tokensNew = append(tokensNew, *token0)
        } else if token0.text == "}" {
            token0.tpe = TokenTypeScopeClose

            tokensNew = append(tokensNew, *token0)
        } else if strings.Count(token0.text, ":=") == 1 {
            tokensNew = append(tokensNew, Token{tpe: TokenTypeName, text: tokenPrev.text})
            tokensNew = append(tokensNew, Token{tpe: TokenTypeDeclAssignment, text: ":="})
            tokensNew = append(tokensNew, Token{tpe: TokenTypeExpr, text: token1.text})
        } else if strings.Count(token0.text, ".") == 1 {
            substrings := strings.Split(token0.text, ".")
            name := substrings[0]
            tokensNew = append(tokensNew, Token{tpe: TokenTypeName, text: name})

            call := substrings[1]
            if strings.Count(call, "(") != strings.Count(call, ")") {
                // read until closing paren
                if i < len(tokens) - 1 {
                    for j := i+1; ; j++ {
                        if j == len(tokens) - 1 {
                            break
                        }
                        token := tokens[j]
                        call += " " + token.text

                        if strings.Contains(token.text, ")") {
                            break
                        }
                    }
                }
            }

            tokensNew = append(tokensNew, Token{tpe: TokenTypeCall, text: call})
        } else if strings.Count(token0.text, ":") > 0 {
            substrings := strings.Split(token0.text, ":")
            name := substrings[0]
            tokensNew = append(tokensNew, Token{tpe: TokenTypeName, text: name})

            tokensNew = append(tokensNew, Token{tpe: TokenTypeType, text: token1.text})
        } else if strings.Count(token0.text, "=") == 1 {
            tokensNew = append(tokensNew, Token{tpe: TokenTypeName, text: tokenPrev.text})
            tokensNew = append(tokensNew, Token{tpe: TokenTypeEquals, text: "="})
            tokensNew = append(tokensNew, Token{tpe: TokenTypeExpr, text: token1.text})
        }
    }

    // debug tokens
    var bb strings.Builder
    for _, token := range tokensNew {
        bb.WriteString(token.tpe.String())
        bb.WriteString(" ")
        bb.WriteString(token.text)
        bb.WriteString(", ")
    }
    fmt.Println(bb.String())


    // go source per declaration
    textGoPerDeclaration := make(map[string]string)
    for i, token0 := range tokensNew {
        if token0.tpe == TokenTypeFunction {
            // tokenFunction := token0

            token1 := tokensNew[i+1]
            if token1.tpe != TokenTypeName {
                panic("expected function name")
            }

            tokenName := token1

            token2 := tokensNew[i+2]
            if token2.tpe != TokenTypeArgs {
                panic("expected function arguments")
            }

            tokenArgs := token2

            var tokensScope []Token
            isInScope := false
            for j := i+3; ; j++ {
                tokenN := tokensNew[j]
                if tokenN.tpe == TokenTypeScopeClose {
                    isInScope = false
                    break
                }
                if isInScope {
                    tokensScope = append(tokensScope, tokenN)
                }
                if tokenN.tpe == TokenTypeScopeOpen {
                    isInScope = true
                }
            }

            var textGoScope strings.Builder
            textGoScope.WriteString("func ")
            textGoScope.WriteString(tokenName.text)
            textGoScope.WriteString(tokenArgs.text)
            textGoScope.WriteString(" {\n")
            for i, token0 := range tokensScope {
                if token0.tpe == TokenTypeName {
                    nameGo := token0.text
                    {
                        if token0.text == "std" {
                            nameGo = "fmt"
                        }
                    }

                    var token1 Token
                    if i < len(tokensScope) -1 {
                        token1 = tokensScope[i+1]
                    } else {
                        token1 = tokenNew()
                    }
                    var token2 Token
                    if i < len(tokensScope) -2 {
                        token2 = tokensScope[i+2]
                    } else {
                        token2 = tokenNew()
                    }

                    if token1.tpe == TokenTypeCall {
                        callGo := token1.text
                        {
                            if token0.text == "std" {
                                callGo = stringUpperFirstChar(callGo)
                            }
                        }
                        textGoScope.WriteString(nameGo)
                        textGoScope.WriteString(".")
                        textGoScope.WriteString(callGo)
                        textGoScope.WriteString("\n")
                    } else if token1.tpe == TokenTypeType {
                        textGoScope.WriteString("var")
                        textGoScope.WriteString(" ")
                        textGoScope.WriteString(nameGo)
                        typeGo := token1.text
                        textGoScope.WriteString(" ")
                        textGoScope.WriteString(typeGo)
                        textGoScope.WriteString("\n")
                    } else if token1.tpe == TokenTypeEquals {
                        textGoScope.WriteString(nameGo)
                        textGoScope.WriteString(" = ")

                        exprGo := token2.text
                        textGoScope.WriteString(exprGo)
                        textGoScope.WriteString("\n")                        
                    } else if token1.tpe == TokenTypeDeclAssignment {
                        textGoScope.WriteString(nameGo)
                        textGoScope.WriteString(" := ")

                        exprGo := token2.text
                        textGoScope.WriteString(exprGo)
                        textGoScope.WriteString("\n")                        
                    }
                }
            }
            textGoScope.WriteString("}")

            scopeName := tokenName.text
            textGoPerDeclaration[scopeName] = textGoScope.String()
        }
    }

    // go source total
    var textGo strings.Builder
    for i, token0 := range tokensNew {
        var token1 Token
        if i < len(tokensNew) -1 {
            token1 = tokensNew[i+1]
        } else {
            token1 = tokenNew()
        }
        if token0.tpe == TokenTypePackage {
            textGo.WriteString("package ")
            textGo.WriteString(token1.text)
            textGo.WriteString("\n")
            textGo.WriteString("\n")
        } else if token0.tpe == TokenTypeImport {
            textGo.WriteString("import ")
            nameGo := token1.text
            {
                if token1.text == "\"std\"" {
                    nameGo = "\"fmt\""
                }
            }

            textGo.WriteString(nameGo)
            textGo.WriteString("\n")
        } else if token0.tpe == TokenTypeFunction {
            textGo.WriteString("\n")
            textGoForDeclaration := textGoPerDeclaration[token1.text]
            textGo.WriteString(textGoForDeclaration)
        }
    }

    // write go source
    fileOut, err := os.Create(filePathOut)
    if err != nil {
        panic(fmt.Errorf("os.Create %w", err))
    }

    textGoStr := textGo.String()
    writerOut := bufio.NewWriter(fileOut)
    _, err = writerOut.WriteString(textGoStr)
    if err != nil {
        panic(fmt.Errorf("WriteString %w", err))
    }
    err = writerOut.Flush()
    if err != nil {
        panic(fmt.Errorf("Flush() %w", err))
    }

    fmt.Printf("Written %v OK\n", filePathOut)
}

func tokenNew() Token {
    return Token{tpe: 0, text: ""}
}

type Token struct {
    tpe TokenType
    text string
}

type TokenType int
const (
    TokenTypeNone = 0
    TokenTypePackage = 1
    TokenTypeName = 2
    TokenTypeImport = 3
    TokenTypeStringLiteral = 4
    TokenTypeFunction = 5
    TokenTypeScopeOpen = 6
    TokenTypeScopeClose = 7
    TokenTypeStatement = 8
    TokenTypeCall = 9
    TokenTypeArgs = 10
    TokenTypeType = 11
    TokenTypeEquals = 12
    TokenTypeExpr = 13
    TokenTypeDeclAssignment = 14
)

var tokenTypeStrings = map[TokenType]string{
    TokenTypeNone: "none",
    TokenTypePackage: "pack",
    TokenTypeName: "name",
    TokenTypeImport: "impr",
    TokenTypeStringLiteral: "str",
    TokenTypeFunction: "func",
    TokenTypeScopeOpen: "open",
    TokenTypeScopeClose: "clos",
    TokenTypeStatement: "stmt",
    TokenTypeCall: "call",
    TokenTypeArgs: "args",
    TokenTypeType: "type",
    TokenTypeEquals: "eqls",
    TokenTypeExpr: "expr",
    TokenTypeDeclAssignment: "dass",
}

func (tt TokenType) String() string {
    return tokenTypeStrings[tt]
}

func stringUpperFirstChar(s string) string {
    r, size := utf8.DecodeRuneInString(s)
    if r == utf8.RuneError && size <= 1 {
        return s
    }
    lc := unicode.ToUpper(r)
    if r == lc {
        return s
    }
    return string(lc) + s[size:]
}