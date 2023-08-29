package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

type (
	Position struct {
		X int `json:"x"`
		Y int `json:"y"`
	}

	Tick struct {
		VelX int `json:"velX"`
		VelY int `json:"velY"`
	}

	GameState struct {
		GameID string   `json:"gameId"`
		Width  int      `json:"width"`
		Height int      `json:"height"`
		Score  int      `json:"score"`
		Fruit  Position `json:"fruit"`
		Snake  Snake    `json:"snake"`
		Ticks  []Tick   `json:"ticks"`
	}

	Snake struct {
		Position
		VelX int `json:"velX"`
		VelY int `json:"velY"`
	}
)

// initializeGame creates a new game with the given board size
func initializeGame(boardSize Position) GameState {
	snake := Snake{
		Position: Position{X: 0, Y: 0},
		VelX:     1,
		VelY:     0,
	}
	fruit := generateRandomPosition(boardSize.X, boardSize.Y)
	gameID := generateGameID()

	return GameState{
		GameID: gameID,
		Width:  boardSize.X,
		Height: boardSize.Y,
		Score:  0,
		Fruit:  fruit,
		Snake:  snake,
		Ticks:  nil,
	}
}

// generateGameID generates a new game ID
func generateGameID() string {
	return fmt.Sprintf("game-%d", time.Now().UnixNano())
}

// isValidMove returns true if the given move is valid
func isValidMove(currentState, nextState GameState) bool {
	velChangeX := nextState.Snake.VelX - currentState.Snake.VelX
	velChangeY := nextState.Snake.VelY - currentState.Snake.VelY

	if (velChangeX == -currentState.Snake.VelX && velChangeY == 0) ||
		(velChangeX == 0 && velChangeY == -currentState.Snake.VelY) {
		return false
	}

	return true
}

// generateRandomPosition generates a random position within the given bounds
func generateRandomPosition(maxX, maxY int) Position {
	return Position{
		X: rand.Intn(maxX),
		Y: rand.Intn(maxY),
	}
}

// newGameHandler creates a new game with the given width and height
func newGameHandler(w http.ResponseWriter, r *http.Request) {
	width := parseQueryParam(r, "w")
	height := parseQueryParam(r, "h")

	if width <= 0 || height <= 0 {
		http.Error(w, "Invalid width or height", http.StatusBadRequest)
		return
	}

	boardSize := Position{X: width, Y: height}
	gameState := initializeGame(boardSize)

	jsonResponse(w, gameState)
}

// validateHandler validates the given game state
func validateHandler(w http.ResponseWriter, r *http.Request) {
	var currentState GameState
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&currentState)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	newGameState, statusCode := validateTicks(currentState)
	jsonResponseWithStatus(w, newGameState, statusCode)
}

// validateTicks validates the given ticks and returns the new game state
func validateTicks(currentState GameState) (GameState, int) {
	if isGameOver(currentState) {
		return currentState, http.StatusTeapot
	}

	if isFruitEaten(currentState) {
		currentState.Score++
		currentState.Fruit = generateRandomPosition(currentState.Width, currentState.Height)
	}

	newGameState := currentState
	for _, tick := range currentState.Ticks {
		newSnake := Snake{
			Position: Position{
				X: newGameState.Snake.X + tick.VelX,
				Y: newGameState.Snake.Y + tick.VelY,
			},
			VelX: tick.VelX,
			VelY: tick.VelY,
		}

		if !isValidMove(newGameState, GameState{Snake: newSnake}) {
			return currentState, http.StatusBadRequest
		}

		newGameState.Snake = newSnake

		if isGameOver(newGameState) {
			return currentState, http.StatusTeapot
		}
	}

	return currentState, http.StatusOK
}

// isGameOver returns true if the snake has hit a wall
func isGameOver(state GameState) bool {
	return state.Snake.X >= state.Width || state.Snake.Y >= state.Height ||
		state.Snake.X < 0 || state.Snake.Y < 0
}

// isFruitEaten returns true if the snake has eaten the fruit
func isFruitEaten(state GameState) bool {
	return state.Snake.X == state.Fruit.X && state.Snake.Y == state.Fruit.Y
}

// parseQueryParam parses the given query parameter from the request
func parseQueryParam(r *http.Request, param string) int {
	values := r.URL.Query()
	val := values.Get(param)
	if val == "" {
		return 0
	}

	parsedVal, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}

	return parsedVal
}

// jsonResponse writes the given response as JSON
func jsonResponse(w http.ResponseWriter, response any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// jsonResponseWithStatus writes the given response as JSON with the given status code
func jsonResponseWithStatus(w http.ResponseWriter, response any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	jsonResponse(w, response)
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/new", newGameHandler)
	r.Post("/validate", validateHandler)

	http.ListenAndServe(":8080", r)
}
