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

    entries, err := os.ReadDir(dir)
    if err != nil {
        panic(fmt.Errorf("ReadDir %w", err))
    }
    for _, entry := range entries {
        name := entry.Name()
        if !strings.Contains(name, ".teo") {
            continue
        }

        var namePrefix string
        {
            suffix := ".teo"
            name, found := strings.CutSuffix(name, suffix)
            if !found {
                panic(fmt.Errorf("should found %v", suffix))
            }
            namePrefix = name
        }

        filePath := filepath.Join(dir, name)
        filePathOut := filepath.Join(dir, namePrefix + ".go")

        fmt.Println("file:", filePath)
        fileBytes, err := os.ReadFile(filePath)
        if err != nil {
            fmt.Println(err)
        }
        fileText := string(fileBytes)
        substrings := strings.Fields(fileText)
        var tokens []Token
        for _, substr := range substrings {
            tokens = append(tokens, Token{tpe: TokenTypeNone, text: substr})
        }

        var tokensNew []Token


        for i := range tokens {
            token0 := &tokens[i]
            var token1 *Token
            if i < len(tokens) -1 {
                token1 = &tokens[i+1]
            } else {
                tkn := tokenNew()
                token1 = &tkn
            }

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
            } else if strings.Count(token0.text, ".") == 1 {
                substrings := strings.Split(token0.text, ".")
                name := substrings[0]
                tokensNew = append(tokensNew, Token{tpe: TokenTypeName, text: name})

                call := substrings[1]
                if i < len(tokens) - 1 {
                    for j := i+1; ; j++ {
                        token := tokens[j]
                        call += " " + token.text

                        if strings.Contains(token.text, ")") {
                            break
                        }
                    }
                }

                tokensNew = append(tokensNew, Token{tpe: TokenTypeCall, text: call})
            }
        }
        var bb strings.Builder
        for _, token := range tokensNew {
            bb.WriteString(token.tpe.String())
            bb.WriteString(" ")
            bb.WriteString(token.text)
            bb.WriteString(", ")
        }
        fmt.Println(bb.String())



        // var substr0 string
        // for _, char := range fileText {
        //     substr0 += string(char)
        //     if unicode.IsSpace(char) {
        //         break
        //     }
        // }
        // substrRest, found := strings.CutPrefix(fileText, substr0)
        // if !found {
        //     panic("should be found")
        // }

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

                        textGoScope.WriteString(nameGo)
                        textGoScope.WriteString(".")

                        var token1 Token
                        if i < len(tokensScope) -1 {
                            token1 = tokensScope[i+1]
                        } else {
                            token1 = tokenNew()
                        }
                        if token1.tpe == TokenTypeCall {
                            nameGo := token1.text
                            {
                                if token0.text == "std" {
                                    nameGo = stringUpperFirstChar(nameGo)
                                }
                            }
                            textGoScope.WriteString(nameGo)
                            textGoScope.WriteString("\n")
                        }
                    }
                }
                textGoScope.WriteString("}")

                scopeName := tokenName.text
                textGoPerDeclaration[scopeName] = textGoScope.String()
            }
        }


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

        fmt.Printf("Written %v\n", filePathOut)
        fmt.Printf("OK\n")
    }
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