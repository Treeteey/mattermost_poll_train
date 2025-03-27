package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/sirupsen/logrus"
	"github.com/tarantool/go-tarantool"
)

type VotingBot struct {
	plugin.MattermostPlugin
	tarantoolConn *tarantool.Connection
	logger        *logrus.Logger
}

type Vote struct {
	ID        string            `json:"id"`
	CreatorID string            `json:"creator_id"`
	Question  string            `json:"question"`
	Options   []string          `json:"options"`
	Votes     map[string]string `json:"votes"`
	IsActive  bool              `json:"is_active"`
	ChannelID string            `json:"channel_id"`
}

func (p *VotingBot) OnActivate() error {
	p.logger = logrus.New()
	p.logger.SetOutput(os.Stdout)
	p.logger.SetLevel(logrus.InfoLevel)

	// Подключение к Tarantool
	conn, err := tarantool.Connect("localhost:3301", tarantool.Opts{
		User: "admin",
		Pass: "admin",
	})
	if err != nil {
		return fmt.Errorf("failed to connect to Tarantool: %v", err)
	}
	p.tarantoolConn = conn

	return nil
}

func (p *VotingBot) OnDeactivate() error {
	if p.tarantoolConn != nil {
		return p.tarantoolConn.Close()
	}
	return nil
}

func (p *VotingBot) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	if !strings.HasPrefix(post.Message, "/vote") {
		return
	}

	args := strings.Fields(post.Message)
	if len(args) < 2 {
		p.replyToPost(post, "Использование: /vote [create|vote|results|end|delete] [параметры]")
		return
	}

	switch args[1] {
	case "create":
		p.handleCreateVote(post, args[2:])
	case "vote":
		p.handleVote(post, args[2:])
	case "results":
		p.handleResults(post, args[2:])
	case "end":
		p.handleEndVote(post, args[2:])
	case "delete":
		p.handleDeleteVote(post, args[2:])
	default:
		p.replyToPost(post, "Неизвестная команда. Используйте: /vote [create|vote|results|end|delete]")
	}
}

func (p *VotingBot) replyToPost(post *model.Post, message string) {
	reply := &model.Post{
		ChannelId: post.ChannelId,
		Message:   message,
		RootId:    post.Id,
	}
	if err := p.API.CreatePost(reply); err != nil {
		p.logger.Errorf("Failed to create reply: %v", err)
	}
}

func (p *VotingBot) handleCreateVote(post *model.Post, args []string) {
	if len(args) < 3 {
		p.replyToPost(post, "Использование: /vote create [вопрос] [вариант1] [вариант2] ...")
		return
	}

	question := args[0]
	options := args[1:]

	vote := Vote{
		ID:        generateVoteID(),
		CreatorID: post.UserId,
		Question:  question,
		Options:   options,
		Votes:     make(map[string]string),
		IsActive:  true,
		ChannelID: post.ChannelId,
	}

	// Сохранение в Tarantool
	_, err := p.tarantoolConn.Insert("votes", []interface{}{vote})
	if err != nil {
		p.logger.Errorf("Failed to save vote: %v", err)
		p.replyToPost(post, "Ошибка при создании голосования")
		return
	}

	message := fmt.Sprintf("Голосование создано! ID: %s\nВопрос: %s\nВарианты ответов:\n", vote.ID, vote.Question)
	for i, option := range vote.Options {
		message += fmt.Sprintf("%d. %s\n", i+1, option)
	}

	p.replyToPost(post, message)
}

func (p *VotingBot) handleVote(post *model.Post, args []string) {
	if len(args) != 2 {
		p.replyToPost(post, "Использование: /vote vote [ID_голосования] [номер_варианта]")
		return
	}

	voteID := args[0]
	optionIndex := args[1]

	// Получение голосования из Tarantool
	var vote Vote
	err := p.tarantoolConn.SelectTyped("votes", "primary", 0, 1, tarantool.IterEq, []interface{}{voteID}, &vote)
	if err != nil {
		p.logger.Errorf("Failed to get vote: %v", err)
		p.replyToPost(post, "Ошибка при получении голосования")
		return
	}

	if !vote.IsActive {
		p.replyToPost(post, "Это голосование уже завершено")
		return
	}

	// Проверка, не голосовал ли уже пользователь
	if _, exists := vote.Votes[post.UserId]; exists {
		p.replyToPost(post, "Вы уже проголосовали в этом голосовании")
		return
	}

	// Сохранение голоса
	vote.Votes[post.UserId] = optionIndex
	_, err = p.tarantoolConn.Update("votes", "primary", []interface{}{voteID}, []interface{}{[]interface{}{"=", "votes", vote.Votes}})
	if err != nil {
		p.logger.Errorf("Failed to update vote: %v", err)
		p.replyToPost(post, "Ошибка при сохранении голоса")
		return
	}

	p.replyToPost(post, "Ваш голос успешно сохранен!")
}

func (p *VotingBot) handleResults(post *model.Post, args []string) {
	if len(args) != 1 {
		p.replyToPost(post, "Использование: /vote results [ID_голосования]")
		return
	}

	voteID := args[0]

	var vote Vote
	err := p.tarantoolConn.SelectTyped("votes", "primary", 0, 1, tarantool.IterEq, []interface{}{voteID}, &vote)
	if err != nil {
		p.logger.Errorf("Failed to get vote: %v", err)
		p.replyToPost(post, "Ошибка при получении результатов")
		return
	}

	message := fmt.Sprintf("Результаты голосования:\nВопрос: %s\n", vote.Question)

	// Подсчет голосов
	voteCount := make(map[string]int)
	for _, option := range vote.Votes {
		voteCount[option]++
	}

	for i, option := range vote.Options {
		count := voteCount[fmt.Sprintf("%d", i+1)]
		message += fmt.Sprintf("%d. %s: %d голосов\n", i+1, option, count)
	}

	p.replyToPost(post, message)
}

func (p *VotingBot) handleEndVote(post *model.Post, args []string) {
	if len(args) != 1 {
		p.replyToPost(post, "Использование: /vote end [ID_голосования]")
		return
	}

	voteID := args[0]

	var vote Vote
	err := p.tarantoolConn.SelectTyped("votes", "primary", 0, 1, tarantool.IterEq, []interface{}{voteID}, &vote)
	if err != nil {
		p.logger.Errorf("Failed to get vote: %v", err)
		p.replyToPost(post, "Ошибка при получении голосования")
		return
	}

	if vote.CreatorID != post.UserId {
		p.replyToPost(post, "Только создатель голосования может его завершить")
		return
	}

	vote.IsActive = false
	_, err = p.tarantoolConn.Update("votes", "primary", []interface{}{voteID}, []interface{}{[]interface{}{"=", "is_active", false}})
	if err != nil {
		p.logger.Errorf("Failed to end vote: %v", err)
		p.replyToPost(post, "Ошибка при завершении голосования")
		return
	}

	p.replyToPost(post, "Голосование завершено!")
}

func (p *VotingBot) handleDeleteVote(post *model.Post, args []string) {
	if len(args) != 1 {
		p.replyToPost(post, "Использование: /vote delete [ID_голосования]")
		return
	}

	voteID := args[0]

	var vote Vote
	err := p.tarantoolConn.SelectTyped("votes", "primary", 0, 1, tarantool.IterEq, []interface{}{voteID}, &vote)
	if err != nil {
		p.logger.Errorf("Failed to get vote: %v", err)
		p.replyToPost(post, "Ошибка при получении голосования")
		return
	}

	if vote.CreatorID != post.UserId {
		p.replyToPost(post, "Только создатель голосования может его удалить")
		return
	}

	_, err = p.tarantoolConn.Delete("votes", "primary", []interface{}{voteID})
	if err != nil {
		p.logger.Errorf("Failed to delete vote: %v", err)
		p.replyToPost(post, "Ошибка при удалении голосования")
		return
	}

	p.replyToPost(post, "Голосование успешно удалено!")
}

func generateVoteID() string {
	return fmt.Sprintf("vote_%d", time.Now().UnixNano())
}
