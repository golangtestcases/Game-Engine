package engine

import (
	"os"
	"time"

	"github.com/hajimehoshi/go-mp3"
	"github.com/hajimehoshi/oto/v2"
)

// AudioPlayer управляет циклическим воспроизведением фоновой музыки.
// Текущая реализация проста: один активный трек, без микширования и кроссфейдов.
type AudioPlayer struct {
	context  *oto.Context
	running  bool
	filepath string
}

// NewAudioPlayer поднимает аудио-контекст (44.1kHz, стерео, 16-bit PCM).
func NewAudioPlayer() (*AudioPlayer, error) {
	ctx, ready, err := oto.NewContext(44100, 2, 2)
	if err != nil {
		return nil, err
	}
	<-ready

	return &AudioPlayer{
		context: ctx,
		running: false,
	}, nil
}

// PlayLoop запускает бесконечное воспроизведение MP3-файла в отдельной горутине.
// При ошибке чтения/декодирования выполняется пауза и повторная попытка.
func (a *AudioPlayer) PlayLoop(filepath string) error {
	if a.running {
		return nil
	}

	a.running = true
	a.filepath = filepath

	go func() {
		for a.running {
			if err := a.playOnce(a.filepath); err != nil {
				time.Sleep(time.Second)
			}
		}
	}()

	return nil
}

// playOnce воспроизводит файл один раз до завершения.
func (a *AudioPlayer) playOnce(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder, err := mp3.NewDecoder(file)
	if err != nil {
		return err
	}

	player := a.context.NewPlayer(decoder)
	defer player.Close()

	player.Play()
	for player.IsPlaying() {
		time.Sleep(time.Millisecond * 100)
	}

	return nil
}

// Stop останавливает цикл воспроизведения.
// Уже созданный player завершится сам после выхода из текущей итерации.
func (a *AudioPlayer) Stop() {
	a.running = false
}

// SetVolume — заглушка: в текущей связке `oto/v2` громкость не регулируется напрямую.
func (a *AudioPlayer) SetVolume(volume float64) {
	_ = volume
}

// Close освобождает ресурсы плеера.
func (a *AudioPlayer) Close() error {
	a.Stop()
	return nil
}

// PlayBackgroundMusic — удобная обертка для быстрого старта фонового трека.
func PlayBackgroundMusic(filepath string) (*AudioPlayer, error) {
	player, err := NewAudioPlayer()
	if err != nil {
		return nil, err
	}

	if err := player.PlayLoop(filepath); err != nil {
		player.Close()
		return nil, err
	}

	return player, nil
}
