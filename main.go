package main

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
	"math"
	"math/rand"
	"net/http"
)

const round = 2

const debug = true

func indexOf(element float64, data []float64) (result []int) {
	for k, v := range data {
		if element == v {
			result = append(result, k)
		}
	}
	return
}

type failLineItem struct {
	Time float64
}

type workLineItem struct {
	StartAt float64
	EndAt   float64
}

type workLine struct {
	Items []workLineItem
}

type queueLineItem struct {
	StartAt float64
	EndAt   float64
}

type queueLine struct {
	Items []queueLineItem
}

func simulate(clients []float64, queuesCount int, throughputStations []int, interval int) (int, int, []workLine, []failLineItem, []queueLine) {
	var served int
	var fail int

	countStations := len(throughputStations)

	// Временная шкала

	// Работа станций
	timelineWork := make([]workLine, countStations)

	// Отказы
	var timelineFail []failLineItem

	// Работа очередей
	timelineQueue := make([]queueLine, queuesCount)

	// Очередь обслуживаемых
	queuesWork := make([][]float64, countStations)

	// Очередь ожидающих
	queuesWait := make([][]float64, queuesCount)

	var moment float64

	if debug {
		fmt.Print("[debug] Клиенты: ", clients, "\n\n")
	}

	var lost int
	for i := 0.0; i < float64(interval); i += 1 / math.Pow10(round) {
		moment = Round(i, round)

		// Освобождаем станции
		for j := 0; j < len(queuesWork); j++ {
			if len(queuesWork[j]) != 0 {
				if debug {
					fmt.Print(moment, " [debug] station #", j, " ", queuesWork[j], "\n")
				}
				if Round(queuesWork[j][0], round) <= moment {
					queuesWork[j] = queuesWork[j][1:]
					served += 1

					if debug {
						fmt.Print(moment, " [debug] Обслужен клиент\n")
					}
				}
			}
		}

		station := getEmptyStation(queuesWork)
		if station != -1 {

			// Продвижение очереди
			for j := 0; j < len(queuesWait); j++ {
				if len(queuesWait[j]) == 0 {
					// Если очередь пуста, то и очереди под ней пусты
					break
				}

				timelineQueue[j].Items[len(timelineQueue[j].Items)-1].EndAt = moment

				// Людей из первой станции нужно отправить на станцию
				if j == 0 {
					// Отправляем на заправку
					endAt := Round(moment+getExpectedTime(throughputStations[station]), round)
					queuesWork[station] = append(queuesWork[station], endAt)
					timelineWork[station].Items = append(timelineWork[station].Items, workLineItem{StartAt: moment, EndAt: endAt})

					// Освобождаем место
				} else {
					queuesWait[j-1] = queuesWait[j]
					timelineQueue[j-1].Items = append(timelineQueue[j-1].Items, queueLineItem{StartAt: moment, EndAt: float64(interval)})
				}
				queuesWait[j] = nil
			}
		}

		// Есть свободная станция

		// Пришел новый клиент
		clientsAtMoment := indexOf(moment, clients)

		queue := -1
		for i := 0; i < len(clientsAtMoment); i++ {
			station = getEmptyStation(queuesWork)
			queue = getEmptyQueue(queuesWait, moment)

			clientIndex := clientsAtMoment[i]

			// Есть клиент
			lost += 1

			if station != -1 {
				// Есть свободная станция
				workEndAt := Round(moment+getExpectedTime(throughputStations[station]), round)

				if debug {
					fmt.Print(moment, " [debug] Отправлен на станцию #", station+1, ": ", moment, " - ", workEndAt, "\n")
				}

				timelineWork[station].Items = append(timelineWork[station].Items, workLineItem{StartAt: moment, EndAt: workEndAt})
				queuesWork[station] = append(queuesWork[station], workEndAt)
			} else if queue != -1 {
				// Есть место в очереди
				queuesWait[queue] = append(queuesWait[queue], clients[clientIndex])
				timelineQueue[queue].Items = append(timelineQueue[queue].Items, queueLineItem{StartAt: moment, EndAt: float64(interval)})

				if debug {
					fmt.Print(moment, " [debug] Отправлен в очередь #", queue, "\n")
				}
				//fail += 1
			} else {
				fail += 1
				timelineFail = append(timelineFail, failLineItem{Time: moment})
				if debug {
					fmt.Print(moment, " [debug] Отказ клиенту \n")
				}
			}
		}
	}

	return served, fail, timelineWork, timelineFail, timelineQueue
}

func getEmptyStation(queuesWork [][]float64) (station int) {
	for i := 0; i < len(queuesWork); i++ {
		// Станция свободна
		if len(queuesWork[i]) == 0 {
			station = i
			return
		}
	}

	// Нет доступных станций
	station = -1
	return
}

func getEmptyQueue(queues [][]float64, time float64) (queue int) {
	for i := 0; i < len(queues); i++ {
		// Ищем свободное место

		// Очередь пуста
		if len(queues[i]) == 0 {
			queue = i
			return
		}
	}
	queue = -1
	return
}

func generateClients(clientPerHour int, interval int) (result []float64) {
	var overload float64
	var currentTime float64

	currentTime = 0
	for i := 0; i < interval; i++ {
		clients, over, clock := generateQueueInHour(clientPerHour, currentTime, overload, i+1 < interval)
		currentTime = clock
		overload = over
		result = append(result, clients...)
	}
	return
}

func generateQueueInHour(clientPerHour int, clock float64, startFrom float64, canOverload bool) (result []float64, overload float64, currentTime float64) {
	var sum float64
	var randomTime float64

	currentTime = clock
	sum = -startFrom
	for {
		randomTime = getExpectedTime(clientPerHour)
		moment := Round(randomTime+currentTime, round)
		if sum+randomTime >= 1 {
			if canOverload == true {
				// Возьем еще одного
				sum += randomTime
				result = append(result, moment)
				currentTime = moment
			}
			break
		} else {
			sum += randomTime
			result = append(result, moment)
			currentTime = moment
		}
	}
	overload = 1 - sum
	return
}

// Round return rounded version of x with prec precision.
//
// Special cases are:
//
//	Round(±0) = ±0
//	Round(±Inf) = ±Inf
//	Round(NaN) = NaN
func Round(x float64, prec int) float64 {
	var rounder float64
	pow := math.Pow(10, float64(prec))
	intermed := x * pow
	_, frac := math.Modf(intermed)
	if frac >= 0.5 {
		rounder = math.Ceil(intermed)
	} else {
		rounder = math.Floor(intermed)
	}

	return rounder / pow
}

func getExpectedTime(alpha int) (result float64) {
	r := rand.Float64() // from 0 to 1
	result = Round(1/float64(alpha)*math.Log(r)*-1, round)
	return
}

type simulateParams struct {
	ClientsPerHour int   `form:"clientsPerHour" binding:"required"`
	Interval       int   `form:"interval"`
	Stations       []int `form:"stations"`
	Queues         int   `form:"queues"`
}

func simulateRoute(c *gin.Context) {
	var params simulateParams

	// Bind the request parameters to the struct
	if err := c.ShouldBind(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create clients
	clients := generateClients(params.ClientsPerHour, params.Interval)

	// Simulate
	served, fail, timeLineWork, timeLineFails, timelineQueue := simulate(clients, params.Queues, params.Stations, params.Interval)

	// Return the simulation results
	c.JSON(http.StatusOK, gin.H{
		"clients":       clients,
		"totalClients":  served + fail,
		"served":        served,
		"fails":         fail,
		"timelineWork":  timeLineWork,
		"timeLineFails": timeLineFails,
		"timelineQueue": timelineQueue,
	})

	fmt.Print("\n\nРезультат")
	fmt.Print("\nЗаявки: ", clients)
	fmt.Print("\nВременая шкала станций: ", timeLineWork)
	fmt.Print("\nВременая шкала отказов: ", timeLineFails)
	fmt.Print("\nВременая шкала очередей: ", timelineQueue)
	fmt.Print("\nКлиентов за день: ", len(clients))
	fmt.Print("\nОбработано заявок: ", served)
	fmt.Print("\nОтказы: ", fail)
	fmt.Print("\nНе успели к концу рабочего дня: ", len(clients)-served-fail)
}

func main() {
	router := gin.Default()

	// Use CORS middleware
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"} // You can adjust this to your specific front-end URL(s)
	config.AllowMethods = []string{"GET", "POST", "OPTIONS"}
	router.Use(cors.New(config))

	router.POST("/simulate", simulateRoute)
	err := router.Run("localhost:7243")
	if err != nil {
		return
	}

	fmt.Print("Расчетная работа\n\n")

	fmt.Print("Режим заполнения 0 - авто, 1 - вручную\n> ")
	var manual bool
	if _, err := fmt.Scan(&manual); err != nil {
		log.Print("Scan error, ", err)
		return
	}

	var clientPerHour int
	if manual {
		fmt.Print("Введите ожидаемое количество клиентов в час\n> ")
		if _, err := fmt.Scan(&clientPerHour); err != nil {
			log.Print("Scan error, ", err)
			return
		}
	} else {
		clientPerHour = 10
	}

	var countStations int
	if manual {
		fmt.Print("Введите количество станции\n> ")
		if _, err := fmt.Scan(&countStations); err != nil {
			log.Print("Scan error, ", err)
			return
		}
	} else {
		countStations = 2
	}

	// Пропускная способность станции
	throughputStations := make([]int, countStations)
	for i := 0; i < countStations; i++ {
		if manual {
			fmt.Print("Пропускная способность станции №", i+1, "\n> ")
			_, err := fmt.Scan(&throughputStations[i])
			if err != nil {
				log.Print("Scan error, ", err)
				return
			}
		} else {
			throughputStations[i] = 5
		}
	}

	// Количество очередей
	var queuesCount int
	if manual {
		fmt.Print("Введите количество мест в очереди\n> ")
		if _, err := fmt.Scan(&queuesCount); err != nil {
			log.Print("Scan error, ", err)
			return
		}
	} else {
		queuesCount = 3
	}

	// Временной интервал симуляции
	var interval int
	if manual {
		fmt.Print("Введите количество эмилируемых часов\n> ")
		if _, err := fmt.Scan(&interval); err != nil {
			log.Print("Scan error, ", err)
			return
		}
	} else {
		interval = 10
	}

	// Создаем заявки на рабочий день
	clients := generateClients(clientPerHour, interval)

	// Симулируем
	served, fail, timeLineWork, timeLineFails, timelineQueue := simulate(clients, queuesCount, throughputStations, interval)

	// Отчет
	fmt.Print("\n\nРезультат")
	fmt.Print("\nЗаявки: ", clients)
	fmt.Print("\nВременая шкала станций: ", timeLineWork)
	fmt.Print("\nВременая шкала отказов: ", timeLineFails)
	fmt.Print("\nВременая шкала очередей: ", timelineQueue)
	fmt.Print("\nКлиентов за день: ", len(clients))
	fmt.Print("\nОбработано заявок: ", served)
	fmt.Print("\nОтказы: ", fail)
	fmt.Print("\nНе успели к концу рабочего дня: ", len(clients)-served-fail)
}
