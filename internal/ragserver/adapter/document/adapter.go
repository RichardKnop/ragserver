package document

import (
	"github.com/neurosnap/sentences"
	"github.com/spf13/viper"
	"google.golang.org/genai"
)

type Adapter struct {
	client              *genai.Client
	training            *sentences.Storage
	generativeModelName string
}

type Option func(*Adapter)

func New(client *genai.Client, training *sentences.Storage, options ...Option) *Adapter {
	a := &Adapter{
		client:              client,
		training:            training,
		generativeModelName: viper.GetString("ai.models.generative"),
	}

	for _, o := range options {
		o(a)
	}

	return a
}
