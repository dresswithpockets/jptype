package main

import (
    "bufio"
    "fmt"
    "math/rand"
    "os"
    "strings"
    "text/scanner"
    "time"
    "unicode"

    "github.com/eiannone/keyboard"
    "github.com/fatih/color"
    jpkana "github.com/gojp/kana"
    "github.com/inancgumus/screen"
)

var inProgressKanaColor = color.New(color.FgMagenta)
var incorrectKanaColor = color.New(color.FgRed)
var tentativeKanaColor = color.New(color.FgYellow)
var inProgressWordColor = color.New(color.Bold)
var correctWordColor = color.New(color.FgGreen)

type KanaState int

const (
    KanaInactive  KanaState = 1 << iota
    KanaActive    KanaState = 1 << iota
    KanaCorrect   KanaState = 1 << iota
    KanaIncorrect KanaState = 1 << iota
)

type Kana struct {
    kana  rune
    state KanaState
}

type StateWord struct {
    kana   []Kana
    typed  bool
    active bool
}

type Stats struct {
    wordsPerMinute float32
    keysPerMinute  float32
}

type State struct {
    words          []string
    sessionWords   []StateWord
    activeIndex    int
    currentInput   string
    clearNecessary bool
    stats          Stats
}

func (state State) getWordsView() string {
    view := ""
    for _, word := range state.sessionWords {
        if word.typed {
            view = fmt.Sprintf("%v  %v", view, correctWordColor.Sprint(word.getNormalString()))
        } else if word.active {
            coloredWord := inProgressWordColor.Sprint(word.getColoredString())
            view = fmt.Sprintf("%v  %v", view, coloredWord)
        } else {
            view = fmt.Sprintf("%v  %v", view, word.getNormalString())
        }
    }
    return view
}

func (state *State) selectRandomWords(count int) {
    rand.Seed(time.Now().UnixNano())
    for i := 0; i < count; i++ {
        word := state.words[rand.Intn(len(state.words))]
        wordKana := make([]Kana, 0, len(word))
        wordScanner := scanner.Scanner{}
        wordScanner.Init(strings.NewReader(word))
        for kana := wordScanner.Next(); kana != scanner.EOF; kana = wordScanner.Next() {
            wordKana = append(wordKana, Kana{
                kana,
                KanaInactive,
            })
        }

        state.sessionWords[i] = StateWord{
            wordKana,
            false,
            false,
        }
    }

    state.sessionWords[0].active = true
    state.sessionWords[0].kana[0].state = KanaActive
}

func (state *State) handleInput(input keyboard.KeyEvent) {
    if input.Rune == 0 {
        switch input.Key {
        case keyboard.KeyBackspace:
            if last := len(state.currentInput) - 1; last >= 0 {
                state.currentInput = state.currentInput[:last]
                state.clearNecessary = true
            }
            break
        }
    } else {
        state.currentInput = fmt.Sprintf("%v%v", state.currentInput, string(input.Rune))
    }

    sessionWord := &state.sessionWords[state.activeIndex]
    challenge := sessionWord.getNormalString()
    jpString := jpkana.RomajiToKatakana(state.currentInput)
    if jpString == challenge {
        state.currentInput = ""
        state.sessionWords[state.activeIndex].active = false
        state.sessionWords[state.activeIndex].typed = true
        state.activeIndex++
        if state.activeIndex < len(state.sessionWords) {
            state.sessionWords[state.activeIndex].active = true
            state.sessionWords[state.activeIndex].kana[0].state = KanaActive
            state.clearNecessary = true
        } else {
            // TODO: handle game completion
        }
    } else {
        wordScanner := scanner.Scanner{}
        wordScanner.Init(strings.NewReader(jpString))
        kanaIndex := 0
        for kana := wordScanner.Next(); kana != scanner.EOF; kana = wordScanner.Next() {
            if sessionWord.kana[kanaIndex].kana == kana {
                sessionWord.kana[kanaIndex].state = KanaCorrect
                kanaIndex++
                sessionWord.kana[kanaIndex].state = KanaActive
            } else if unicode.Is(unicode.Katakana, kana) {
                sessionWord.kana[kanaIndex].state = KanaIncorrect
                break
            } else {
                sessionWord.kana[kanaIndex].state = KanaActive
                break
            }
        }
    }
}

func (state State) getInputView() string {
    return jpkana.RomajiToKatakana(state.currentInput)
}

func (word StateWord) getColoredString() string {
    if word.typed {
        return correctWordColor.Sprintf(word.getNormalString())
    }
    str := ""
    for _, kana := range word.kana {
        str = fmt.Sprintf("%v%v", str, kana.getColoredString())
    }
    return str
}

func (word StateWord) getNormalString() string {
    b := make([]rune, 0, 32)
    for _, kana := range word.kana {
        b = append(b, kana.kana)
    }
    return string(b)
}

func (kana Kana) getColoredString() string {
    coloredRune := string(kana.kana)
    switch kana.state {
    case KanaActive:
        coloredRune = inProgressKanaColor.Sprint(coloredRune)
        break
    case KanaCorrect:
        coloredRune = tentativeKanaColor.Sprint(coloredRune)
        break
    case KanaIncorrect:
        coloredRune = incorrectKanaColor.Sprint(coloredRune)
    }
    return coloredRune
}

func getWords() []string {
    wordsFile, err := os.Open("./words.txt")
    if err != nil {
        panic(err)
    }
    defer wordsFile.Close()

    wordsScanner := bufio.NewScanner(wordsFile)
    words := make([]string, 0, 64)
    for wordsScanner.Scan() {
        words = append(words, wordsScanner.Text())
    }
    return words
}

func main() {

    fmt.Println("Reading configuration...")
    state := State{
        getWords(),
        make([]StateWord, 16),
        0,
        "",
        false,
    }

    state.selectRandomWords(16)

    keysEvents, err := keyboard.GetKeys(10)
    if err != nil {
        panic(err)
    }
    defer keyboard.Close()

    screen.Clear()

    for {
        screen.MoveTopLeft()

        fmt.Fprintln(color.Output, "\n", state.getWordsView())
        fmt.Fprint(color.Output, "\n...入力：", state.getInputView())

        if state.clearNecessary {
            fmt.Print("                                        ")
            state.clearNecessary = false
            continue
        }

        event := <-keysEvents
        if event.Err != nil {
            panic(event.Err)
        }

        state.handleInput(event)
    }
}
