package main

import (
	"fmt"
	"strings"
	"time"
)

type clientContext struct {
	servConn *serverConnection
	channel  string
	cb       *chatBox
}

type clientCommand func(ctx *clientContext, args ...string)

var clientCommands map[string]clientCommand

func init() {
	clientCommands = map[string]clientCommand{
		"test":   testCmd,
		"me":     meCmd,
		"join":   joinCmd,
		"part":   partCmd,
		"notice": noticeCmd,
		"msg":    privmsgCmd,
		"nick":   nickCmd,
		"quit":   quitCmd,
		"mode":   modeCmd,
		"clear":  clearCmd,
	}
}

func testCmd(ctx *clientContext, args ...string) {
}

func meCmd(ctx *clientContext, args ...string) {
	msg := strings.Join(args, " ")
	if len(args) == 0 {
		ctx.cb.printMessage("usage: /me [message...]")
		return
	}
	ctx.servConn.conn.Action(ctx.channel, msg)
	ctx.cb.printMessage(fmt.Sprintf("%s * %s %s", time.Now().Format("15:04"), ctx.servConn.cfg.Nick, msg))
}

func joinCmd(ctx *clientContext, args ...string) {
	if len(args) != 1 || len(args[0]) < 2 || args[0][0] != '#' {
		ctx.cb.printMessage("usage: /join [#channel]")
		return
	}
	ctx.servConn.join(args[0])
}

func partCmd(ctx *clientContext, args ...string) {
	ctx.servConn.part(ctx.channel, strings.Join(args, " "))
}

func noticeCmd(ctx *clientContext, args ...string) {
	if len(args) < 2 {
		ctx.cb.printMessage("usage: /notice [#channel or nick] [message...]")
		return
	}
	msg := strings.Join(args[1:], " ")
	ctx.servConn.conn.Notice(args[0], msg)
	ctx.cb.printMessage(fmt.Sprintf("%s *** %s: %s", time.Now().Format("15:04"), ctx.servConn.cfg.Nick, msg))
}

func privmsgCmd(ctx *clientContext, args ...string) {
	if len(args) < 2 || args[0][0] == '#' {
		ctx.cb.printMessage("usage: /msg [nick] [message...]")
		return
	}
	nick := args[0]
	msg := strings.Join(args[1:], " ")

	ctx.servConn.conn.Privmsg(nick, msg)

	cb := ctx.servConn.getChatBox(nick)
	if cb == nil {
		cb = ctx.servConn.createChatBox(nick, CHATBOX_PRIVMSG)
	}
	cb.printMessage(fmt.Sprintf("%s <%s> %s", time.Now().Format("15:04"), ctx.servConn.cfg.Nick, msg))
}

func nickCmd(ctx *clientContext, args ...string) {
	if len(args) != 1 {
		ctx.cb.printMessage("usage: /nick [new nick]")
		return
	}
	ctx.servConn.conn.Nick(args[0])
}

func quitCmd(ctx *clientContext, args ...string) {
	ctx.servConn.conn.Quit(strings.Join(args, " "))
	mw.Close()
}

func modeCmd(ctx *clientContext, args ...string) {
	if len(args) < 2 {
		ctx.cb.printMessage("usage: /mode [#channel or your nick] [mode] [nicks...]")
		return
	}
	ctx.servConn.conn.Mode(args[0], args[1:]...)
}

func clearCmd(ctx *clientContext, args ...string) {
	ctx.cb.textBuffer.SetText("")
}
