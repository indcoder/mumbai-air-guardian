package gemma

import (
	"context"

	"mumbai-air-guardian/backend/internal/evidence"
)

const (
	English = "en"
	Marathi = "mr"
	Hindi   = "hi"
)

type Explanation struct {
	Summary           string   `json:"summary"`
	KeySignals        []string `json:"keySignals"`
	RecommendedAction string   `json:"recommendedAction"`
	Limitations       string   `json:"limitations"`
	Language          string   `json:"language"`
}

type Result struct {
	Mode         string      `json:"mode"`
	ReasonCode   string      `json:"reasonCode,omitempty"`
	EvidenceHash string      `json:"evidenceHash"`
	Explanation  Explanation `json:"explanation"`
}

type CapabilityStatus struct {
	Status             string   `json:"status"`
	Reason             string   `json:"reason,omitempty"`
	Model              string   `json:"model"`
	Languages          []string `json:"languages"`
	GeneratedLanguages []string `json:"generatedLanguages"`
}

type Explainer interface {
	Explain(context.Context, evidence.Activity, string) (Result, error)
	Status(context.Context) CapabilityStatus
}

func TemplateResult(language, reasonCode, evidenceHash string) Result {
	template, ok := templates[language]
	if !ok {
		language = English
		template = templates[English]
	}
	return Result{
		Mode: "template", ReasonCode: reasonCode, EvidenceHash: evidenceHash,
		Explanation: Explanation{
			Summary: template.summary, KeySignals: append([]string(nil), template.keySignals...),
			RecommendedAction: template.recommendedAction,
			Limitations:       template.limitations, Language: language,
		},
	}
}

type localizedTemplate struct {
	summary           string
	keySignals        []string
	recommendedAction string
	limitations       string
}

var templates = map[string]localizedTemplate{
	English: {
		summary: "The suggested period has the lowest mean among the complete hourly forecast windows in the selected range.",
		keySignals: []string{
			"The suggested period has complete, contiguous hourly forecast coverage.",
			"Source-flagged observations do not change this forecast-only calculation.",
			"Forecast issuance time is unavailable and uncertainty is not quantified.",
		},
		recommendedAction: "Use the Go-computed period only as a relative forecast comparison; this is not a health guarantee or medical advice.",
		limitations:       "This explanation uses a fixed forecast snapshot, not live conditions. Forecast issuance time is unavailable and uncertainty is not quantified.",
	},
	Marathi: {
		summary: "निवडलेल्या वेळेच्या मर्यादेतील सलग तासिक अंदाज असलेल्या कालखंडांपैकी सुचवलेल्या कालखंडाची सरासरी सर्वात कमी आहे.",
		keySignals: []string{
			"सुचवलेल्या कालखंडासाठी प्रत्येक तासाचा अंदाज उपलब्ध आहे.",
			"निरीक्षणांवरील गुणवत्ता-चिन्हे या अंदाजाधारित गणनेत बदल करत नाहीत.",
			"अंदाज जारी केल्याची वेळ उपलब्ध नाही आणि अनिश्चितता मोजलेली नाही.",
		},
		recommendedAction: "Go ने मोजलेला कालखंड केवळ अंदाजांची तुलनात्मक माहिती म्हणून वापरा; ही आरोग्याची हमी किंवा वैद्यकीय सल्ला नाही.",
		limitations:       "हे स्पष्टीकरण स्थिर अंदाज-स्नॅपशॉटवर आधारित आहे, प्रत्यक्ष वेळेतील माहितीवर नाही. अंदाज जारी केल्याची वेळ उपलब्ध नाही आणि अनिश्चितता मोजलेली नाही.",
	},
	Hindi: {
		summary: "चुनी गई समय-सीमा में पूरी, लगातार प्रति-घंटा पूर्वानुमान वाली अवधियों में सुझाई गई अवधि का औसत सबसे कम है।",
		keySignals: []string{
			"सुझाई गई अवधि के हर घंटे के लिए पूर्वानुमान उपलब्ध है।",
			"अवलोकनों पर लगे गुणवत्ता-चिह्न इस पूर्वानुमान-आधारित गणना को नहीं बदलते।",
			"पूर्वानुमान जारी होने का समय उपलब्ध नहीं है और अनिश्चितता का मापन नहीं किया गया है।",
		},
		recommendedAction: "Go द्वारा निकाली गई अवधि को केवल पूर्वानुमानों की तुलनात्मक जानकारी के रूप में उपयोग करें; यह स्वास्थ्य की गारंटी या चिकित्सकीय सलाह नहीं है।",
		limitations:       "यह स्पष्टीकरण स्थिर पूर्वानुमान स्नैपशॉट पर आधारित है, लाइव स्थितियों पर नहीं। पूर्वानुमान जारी होने का समय उपलब्ध नहीं है और अनिश्चितता का मापन नहीं किया गया है।",
	},
}
