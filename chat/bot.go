package chat

import (
	"strings"
)

func botReply(content string) string {
	lower := strings.ToLower(strings.TrimSpace(content))

	switch {
	case strings.Contains(lower, "hello") || strings.Contains(lower, "hi"):
		return "Hello! How can I help you today?"
	case strings.Contains(lower, "price") || strings.Contains(lower, "cost"):
		return "Our team can share pricing details. Could you tell me which product or plan you're interested in?"
	case strings.Contains(lower, "refund") || strings.Contains(lower, "complaint"):
		return "I'm sorry to hear that. I'll note your complaint and a human agent will follow up as soon as one is available."
	case strings.Contains(lower, "thank"):
		return "You're welcome! Is there anything else I can help with?"
	case strings.Contains(lower, "bye") || strings.Contains(lower, "goodbye"):
		return "Goodbye! Feel free to reach out anytime."
	default:
		return "Thanks for your message. I've shared this with our support team. A specialist will assist you shortly."
	}
}
