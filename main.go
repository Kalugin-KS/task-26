// Напишите код, реализующий пайплайн, работающий с целыми числами и состоящий из следующих стадий:
// -Стадия фильтрации отрицательных чисел (не пропускать отрицательные числа).
// -Стадия фильтрации чисел, не кратных 3 (не пропускать такие числа), исключая также и 0.
// -Стадия буферизации данных в кольцевом буфере с интерфейсом, соответствующим тому, который был дан в качестве задания в 19 модуле.
// В этой стадии предусмотреть опустошение буфера (и соответственно, передачу этих данных, если они есть, дальше) с определённым интервалом во времени.
// Значения размера буфера и этого интервала времени сделать настраиваемыми (как мы делали: через константы или глобальные переменные).

package main

import (
	"bufio"
	"container/ring"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

type RingBuffer struct {
	r   *ring.Ring
	pos int
	mu  sync.Mutex
}

// Размер кольцевого буфера
var lengthOfBuffer int = 10

// Задержка после которой происходит опустошение буфера, если новые данные больше не вводились
const delay time.Duration = 5 // секунды

func main() {

	fmt.Println("Для завершения работы введите - 'exit'")
	LOG_FILE_PATH := os.Getenv("LOG_FILE_PATH")
	if LOG_FILE_PATH != "" {
		log.SetOutput(&lumberjack.Logger{
			Filename:   LOG_FILE_PATH,
			MaxSize:    200,
			MaxBackups: 3,
			MaxAge:     3,
			Compress:   true,
		})
	}

	stop := make(chan int)  // канал для уведомления о закрытии программы
	input := make(chan int) // входной канал данных
	go read(input, stop)

	chNeg := make(chan int)
	go nonNegative(input, chNeg) // фильтр отрицательных чисел

	chMul := make(chan int)
	go multiOfThree(chNeg, chMul) // фильтр чисел, не кратных 3

	chBuffer := make(chan int)
	go bufferStage(chMul, chBuffer) // стадия записи данных в буфер и получения данных из буфера

	for {
		select {
		case data := <-chBuffer:
			fmt.Printf("Получены данные: %v\n", data)
		case <-stop:
			return
		}
	}
}

// Создание нового кольцевого буфера
func NewRingBuffer(n int) RingBuffer {
	return RingBuffer{ring.New(n), -1, sync.Mutex{}}
}

// Получение размера буфера
func (buf *RingBuffer) Size() int {
	return buf.r.Len()
}

// Запись данных в буфер
func (buf *RingBuffer) Push(el int) {
	defer buf.mu.Unlock()
	buf.mu.Lock()

	if buf.pos == buf.Size()-1 {
		buf.r = buf.r.Next()
		buf.r.Prev().Value = el
	} else {
		buf.pos++
		buf.r.Value = el
		buf.r = buf.r.Next()
	}
}

// Получение данных из буфера
func (buf *RingBuffer) Get() []any {
	if buf.pos < 0 {
		return nil
	}

	buf.mu.Lock()
	defer buf.mu.Unlock()

	out := make([]any, 0, lengthOfBuffer)
	buf.r = buf.r.Move(lengthOfBuffer - buf.pos - 1)
	for i := 0; i <= buf.pos; i++ {
		out = append(out, buf.r.Value)
		buf.r = buf.r.Next()
	}

	buf.pos = -1
	return out
}

// Чтение входных данных из консоли
func read(input chan int, stop chan int) {
	scan := bufio.NewScanner(os.Stdin)

	for scan.Scan() {
		text := scan.Text()
		if text == "exit" {
			fmt.Println("Работа завершена!")
			stop <- 1
			break
		}

		num, err := strconv.Atoi(text)
		if err != nil {
			fmt.Println("введите положительное или отрицательное целое число")
		} else {
			input <- num
		}
	}
}

// Функция фильтрации отрицательных чисел
func nonNegative(input chan int, chNeg chan int) {
	for i := range input {
		if i > 0 {
			chNeg <- i
		}
	}
}

// Функция фильтрации чисел не кратных 3
func multiOfThree(chNeg chan int, chMul chan int) {
	for {
		x := <-chNeg
		if math.Mod(float64(x), 3) == 0 {
			chMul <- int(x)
		}
	}
}

// Функция записи данных в буфер и получения из него
func bufferStage(chMul chan int, chBuffer chan int) {
	rb := NewRingBuffer(lengthOfBuffer)
	for {
		select {
		case data := <-chMul:
			rb.Push(data)

		case <-time.After(time.Duration(delay * time.Second)):
			bufferData := rb.Get()
			for _, data := range bufferData {
				chBuffer <- data.(int)
			}
		}
	}
}
