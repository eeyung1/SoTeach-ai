package tutor

// This file holds the learner-facing content for every supported topic, one
// per-band bundle per topic (README §4 / Agent.md §13 — age calibration is a
// product and safety requirement). Each topic is a map from the three grade
// bands to a topicContent bundle in that band's voice, mirroring Addition's
// shape: diagnostic prompt, explanation, completion, 3 practice questions, and
// 1 transfer question, all with deterministic single-string expected answers
// so checking stays exact-match (session.CheckAnswer).
//
// The registry below is what contentFor in turn.go consults. Curriculum()
// must stay in sync with it (a test enforces this).

var subtractionByBand = map[string]topicContent{
	"Primary 4-6": {
		diagnosticPrompt: "Let's practise taking away. First, tell me in your own words: can you already take small numbers away, like taking 2 away from 5?",
		explanation:      "Taking away means removing some from a group and counting what is left. If you have 8 sweets and you eat 3, count what is left: you have 5 sweets.",
		completion:       "That's right! You have mastered subtracting numbers.",
		practice: []qa{
			{question: "You have 9 sweets and you eat 4. How many sweets are left?", expectedAnswer: "5"},
			{question: "There are 6 eggs in a basket and you use 2 to cook. How many eggs are left?", expectedAnswer: "4"},
			{question: "You have 10 pencils and you give 3 to a friend. How many pencils do you have now?", expectedAnswer: "7"},
		},
		transfer: qa{question: "Mummy bakes 12 buns and gives 5 to the neighbours. How many buns does she have left?", expectedAnswer: "7"},
	},
	"JSS1-3": {
		diagnosticPrompt: "What do you already know about subtraction?",
		explanation:      "Subtraction finds the difference between two numbers: it takes one number away from another to find what is left. In 9 - 4 = 5, 9 is the minuend, 4 the subtrahend, and 5 the difference.",
		completion:       "That's right! You have mastered Subtraction.",
		practice: []qa{
			{question: "What is 9 - 4?", expectedAnswer: "5"},
			{question: "What is 15 - 6?", expectedAnswer: "9"},
			{question: "What is 20 - 8?", expectedAnswer: "12"},
		},
		transfer: qa{question: "What is 23 - 15?", expectedAnswer: "8"},
	},
	"SSS1-3": {
		diagnosticPrompt: "Before we proceed, explain in your own words what subtraction means and how the minuend, subtrahend, and difference relate.",
		explanation:      "Subtraction is the binary operation that finds the difference between two numbers: in a - b = c, a is the minuend, b the subtrahend, and c the difference. Subtraction is not commutative: 9 - 4 does not equal 4 - 9.",
		completion:       "Correct. You have demonstrated mastery of subtraction.",
		practice: []qa{
			{question: "Evaluate: 15 - 6.", expectedAnswer: "9"},
			{question: "Find the difference between 23 and 8.", expectedAnswer: "15"},
			{question: "Compute: 50 - 27.", expectedAnswer: "23"},
		},
		transfer: qa{question: "Determine the value of 100 - 64.", expectedAnswer: "36"},
	},
}

var multiplicationByBand = map[string]topicContent{
	"Primary 4-6": {
		diagnosticPrompt: "Let's practise multiplication. First, tell me in your own words: can you already count things in groups, like how many legs four dogs have?",
		explanation:      "Multiplication is adding the same group again and again. If 3 bags each hold 4 sweets, you have 4 + 4 + 4, which is 12 sweets — that is 3 groups of 4.",
		completion:       "That's right! You have mastered multiplying numbers.",
		practice: []qa{
			{question: "There are 4 bags with 3 sweets in each bag. How many sweets are there altogether?", expectedAnswer: "12"},
			{question: "A box has 5 pencils. How many pencils are in 3 boxes?", expectedAnswer: "15"},
			{question: "A spider has 8 legs. How many legs do 2 spiders have?", expectedAnswer: "16"},
		},
		transfer: qa{question: "There are 6 rows of seats and 4 seats in each row. How many seats are there altogether?", expectedAnswer: "24"},
	},
	"JSS1-3": {
		diagnosticPrompt: "What do you already know about multiplication?",
		explanation:      "Multiplication is repeated addition: 4 × 3 means four groups of three, which totals 12.",
		completion:       "That's right! You have mastered Multiplication.",
		practice: []qa{
			{question: "What is 6 × 7?", expectedAnswer: "42"},
			{question: "What is 8 × 6?", expectedAnswer: "48"},
			{question: "What is 9 × 7?", expectedAnswer: "63"},
		},
		transfer: qa{question: "What is 13 × 5?", expectedAnswer: "65"},
	},
	"SSS1-3": {
		diagnosticPrompt: "Before we proceed, explain in your own words what multiplication is and state the properties you know, such as commutativity and identity.",
		explanation:      "Multiplication is a binary operation that combines two numbers, called factors, into a single product. It is commutative (6 × 7 = 7 × 6) and has 1 as its identity element.",
		completion:       "Correct. You have demonstrated mastery of multiplication.",
		practice: []qa{
			{question: "Evaluate: 12 × 8.", expectedAnswer: "96"},
			{question: "Find the product of 15 and 6.", expectedAnswer: "90"},
			{question: "Compute: 25 × 4.", expectedAnswer: "100"},
		},
		transfer: qa{question: "Determine the product of 17 and 9.", expectedAnswer: "153"},
	},
}

var divisionByBand = map[string]topicContent{
	"Primary 4-6": {
		diagnosticPrompt: "Let's practise sharing out equally. First, tell me in your own words: can you already share sweets fairly between two friends?",
		explanation:      "Sharing equally means putting items into equal groups. If you share 12 sweets among 3 friends, each friend gets 4 sweets.",
		completion:       "That's right! You have mastered sharing numbers.",
		practice: []qa{
			{question: "You share 12 sweets equally among 3 friends. How many sweets does each friend get?", expectedAnswer: "4"},
			{question: "A farmer puts 15 oranges equally into 3 baskets. How many oranges are in each basket?", expectedAnswer: "5"},
			{question: "You share 14 apples equally among 7 friends. How many apples does each friend get?", expectedAnswer: "2"},
		},
		transfer: qa{question: "There are 20 biscuits and 4 children share them equally. How many biscuits does each child get?", expectedAnswer: "5"},
	},
	"JSS1-3": {
		diagnosticPrompt: "What do you already know about division?",
		explanation:      "Division shares a number into equal groups: 24 ÷ 6 asks how many sixes are in 24, which is 4.",
		completion:       "That's right! You have mastered Division.",
		practice: []qa{
			{question: "What is 24 ÷ 6?", expectedAnswer: "4"},
			{question: "What is 45 ÷ 5?", expectedAnswer: "9"},
			{question: "What is 56 ÷ 8?", expectedAnswer: "7"},
		},
		transfer: qa{question: "What is 72 ÷ 9?", expectedAnswer: "8"},
	},
	"SSS1-3": {
		diagnosticPrompt: "Before we proceed, explain in your own words what division means and how the dividend, divisor, and quotient relate.",
		explanation:      "Division distributes a number, the dividend, into equal parts of a given size, the divisor, to produce the quotient: in 72 ÷ 9 = 8, 72 is the dividend, 9 the divisor, and 8 the quotient.",
		completion:       "Correct. You have demonstrated mastery of division.",
		practice: []qa{
			{question: "Evaluate: 144 ÷ 12.", expectedAnswer: "12"},
			{question: "Find the quotient when 126 is divided by 6.", expectedAnswer: "21"},
			{question: "Compute: 225 ÷ 15.", expectedAnswer: "15"},
		},
		transfer: qa{question: "Determine the value of 540 ÷ 18.", expectedAnswer: "30"},
	},
}

// partsOfSpeechByBand covers English Language's first topic. Every question is
// closed-form with exactly one target word of the requested class in the
// sentence, so the expected answer is a single exact word and checking stays
// deterministic. Calibration is in the sentences and the word class asked, not
// the mechanism (Primary: simple nouns/verbs/adjectives; JSS: longer mixed
// sentences plus adverbs; SSS: function-in-context including conjunctions).
var partsOfSpeechByBand = map[string]topicContent{
	"Primary 4-6": {
		diagnosticPrompt: "Let's practise naming words. First, tell me in your own words: do you know what a naming word (a noun) and a doing word (a verb) are?",
		explanation:      "A noun is the name of a person, place, or thing, like 'cat' or 'Ade'. A verb is a doing word, like 'run' or 'cooks'. Describing words like 'big' and 'red' are called adjectives.",
		completion:       "That's right! You have mastered naming words.",
		practice: []qa{
			{question: "Type the doing word (verb) in this sentence: 'Ade runs fast.'", expectedAnswer: "runs"},
			{question: "Type the naming word (noun) in this sentence: 'The cat is sleeping.'", expectedAnswer: "cat"},
			{question: "Type the describing word (adjective) in this sentence: 'The big dog barked.'", expectedAnswer: "big"},
		},
		transfer: qa{question: "Type the verb in this sentence: 'Mummy cooks rice.'", expectedAnswer: "cooks"},
	},
	"JSS1-3": {
		diagnosticPrompt: "What do you already know about the parts of speech, such as nouns, verbs, and adjectives?",
		explanation:      "Every word in a sentence plays a role: nouns name people, places, or things; verbs show actions; adjectives describe nouns; and adverbs describe verbs. Naming a word's role is giving its part of speech.",
		completion:       "That's right! You have mastered the parts of speech.",
		practice: []qa{
			{question: "Type the verb in this sentence: 'The diligent student completed her assignment.'", expectedAnswer: "completed"},
			{question: "Type the noun in this sentence: 'She reads a book quickly.'", expectedAnswer: "book"},
			{question: "Type the adjective in this sentence: 'The tall building stood by the river.'", expectedAnswer: "tall"},
		},
		transfer: qa{question: "Type the adverb in this sentence: 'The dog barked loudly at night.'", expectedAnswer: "loudly"},
	},
	"SSS1-3": {
		diagnosticPrompt: "Before we proceed, explain in your own words what a part of speech is, and show with an example how a word's function in a sentence determines its class.",
		explanation:      "A part of speech is the grammatical category a word belongs to as it is used in a sentence. Function determines class: in 'the light shines', 'light' is a noun, but in 'light the lamp' it is a verb. Context decides the category.",
		completion:       "Correct. You have demonstrated mastery of the parts of speech.",
		practice: []qa{
			{question: "Identify the verb in this sentence and type it: 'The committee finally approved the proposal.'", expectedAnswer: "approved"},
			{question: "Identify the adjective in this sentence and type it: 'An obsolete law was repealed by the senate.'", expectedAnswer: "obsolete"},
			{question: "Identify the adverb in this sentence and type it: 'She spoke persuasively during the debate.'", expectedAnswer: "persuasively"},
		},
		transfer: qa{question: "Identify the conjunction in this sentence and type it: 'We waited because the rain had not stopped.'", expectedAnswer: "because"},
	},
}

// contentByTopic is the registry contentFor consults: topic name -> per-band
// bundle. Topic names are globally unique across subjects, which is what lets
// content stay subject-agnostic (a test enforces this invariant).
var contentByTopic = map[string]map[string]topicContent{
	"Addition":        additionByBand,
	"Subtraction":     subtractionByBand,
	"Multiplication":  multiplicationByBand,
	"Division":        divisionByBand,
	"Parts of Speech": partsOfSpeechByBand,
}
