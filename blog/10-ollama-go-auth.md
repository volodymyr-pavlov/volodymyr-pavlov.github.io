# Ollama in a Go Auth Service

This article describes a login service written in Go: <https://github.com/volodymyr-pavlov/go-vault-webauthn-ollama>. A
user signs in with a passkey, or with a password and a one-time code. Fixed rules give every login a risk score from 0
to 100. A small AI model, `llama3.2:3b`, runs on the same computer through Ollama and gives a second opinion on the
login.

A login is a security decision, and an AI model is a poor base for one. It answers slowly, it can be unavailable, and
its answer can be wrong or in the wrong format. The service therefore limits what the model can do. The model can only
make a decision stricter. It never receives text that a user typed. The service checks every answer before it uses it.
If the model fails, the rules decide alone, and the user still logs in or is still refused in the usual way.

Below, each control comes with its reason. A test follows. It shows that the controls work, but not that the model is
useful.

## Before the Model

Three layers work before the model, and none of them needs it. The security of the login does not depend on a component
that can fail or be tricked.

- **Passkeys.** A passkey is a login method with a secret key that stays on the user's device. The service uses the
  library go-webauthn. It checks the address of the website: a passkey does not work on a fake site. It requires that
  the device verifies the user, for example with a fingerprint. It also watches a counter that the device increases at
  every login. A counter that does not grow means that the passkey may be a copy, and the service disables it. The
  exception is a synced passkey, which always reports zero. An old challenge is useless: it can be used once and expires
  after two minutes.
- **Signed requests.** The service uses no tokens and no cookies. A token can be stolen and used by someone else, but a
  private key that stays in the browser cannot be copied. The browser keeps a private key that nobody can export, and it
  signs every request (RFC 9421). The signature covers the method, the path, the content, the time, a one-time number
  and the session. The service remembers each one-time number for two minutes and refuses a second use. A request also
  needs a time within one minute of the time of the server, which rejects old copies.
- **Vault.** Vault is a program that keeps secrets and keys. The service never keeps the key to the one-time-code
  secrets itself. Vault encrypts the one-time-code secret of each user with a separate key for each user. A secret that
  is copied to another user does not decrypt. Vault also makes the hash that replaces the user in the prompt. A user can
  also store one-time codes for other services. Vault makes these codes itself. The service can ask for a code, but it
  can never read the secret or list the keys.

## Signals and Score

A signal is one fact about a login that can mean a risk. Each signal adds fixed points to the score. Each failed attempt
and each wrong code adds 5 points. The service stores the signals as bits of one number, which keeps the check fast and
the data small.

```go
func (s Signals) Score() uint8 {
	score := 5 * (s.FailedAttempts() + s.TOTPFailures())
	for set := s & signalMask; set != 0; set &= set - 1 {
		score += signalWeights[bits.TrailingZeros16(uint16(set))]
	}

	return min(score, 100)
}
```

The signals are:

- a new device (25 points);
- a factor that was added in the last 24 hours (15 points), because an attacker who takes over a user can add a factor
  of their own;
- an unusual hour for the user (10 points);
- impossible travel (40 points): the speed between the places of two logins is above 1000 km/h, and one person cannot
  make both logins;
- a shared address (10 points): five different users tried to log in from one address in one hour, which can mean an
  attack with stolen passwords;
- an address from the operator's list of VPN and hosting ranges (15 points). The weight is low, because many honest
  users use a VPN. The signal matters mostly together with other signals.

A score below 30 allows the login. A score from 30 to 69 limits it: the user can work, but for a sensitive action, such
as removing a passkey or deleting the data, the user must sign in again first. A score of 70 or more refuses the login.

## The Model Can Only Make a Decision Stricter

The service first calculates the decision from the score. If the rules refused the login, the service does not ask the
model at all, because the answer would change nothing. Otherwise, the model returns a risk level. If this level is
higher than the level of the rules, the decision becomes stricter by one step:

```go
if advice.Level > level {
	if outcome.Decision == DecisionAllow {
		outcome.Decision = DecisionLimit
	} else {
		outcome.Decision = DecisionReview
	}
}
```

"Allow" becomes "limit", and "limit" becomes "review". A review is a limited login with an extra mark on the session.
The mark is meant for a person to check, but no code in the service reads it yet. No path leads to a refusal or to a
weaker decision. This is the main safety rule. Even if someone tricks the model into a wrong answer, the answer can only
add a check, and it can never let an attacker in. After an error, a late answer or a wrong answer, the decision of the
rules stays. The worst result of a bad answer is one extra check for one honest user.

## What the Model Receives and Returns

The prompt is the text that the service sends to the model. A model follows instructions in its text. A user could
therefore try to write instructions into a field, for example into a name. The prompt is therefore a fixed template that
has only values that the service creates itself: the score and the level from the rules, a pseudonym of the user, a
country, a four-hour part of the day, the login method, the names of the active signals, and two counters. The pseudonym
is a part of a hash, and the model never sees the real user. The prompt has no email, no address, no name and no text
that a user typed. A user cannot put instructions into it.

The service asks Ollama to answer in JSON, with a temperature of zero. The temperature controls the randomness of a
model. At zero, the same input usually gives the same answer. The JSON mode asks for a format, but it does not force the
model to follow it. The service does not trust the answer. The function `ParseAdvice` checks it. The function refuses an
answer in each of these cases:

- the answer is longer than 4 KiB, or it is not one JSON object;
- the answer has a field other than `risk_level`, `reasons` and `summary`;
- the risk level is not `low`, `medium` or `high`;
- there are more than five reasons, or a reason is not in a fixed list of eight names;
- the summary is longer than 200 characters, or it has a control character.

A refused answer counts as no advice, and the rules decide alone.

## When the Model Fails

The model runs on a CPU and answers in a few seconds, and the user waits for it. The service must protect the login from
a slow or broken model:

- Each request has a time limit of 8 seconds. A slow model cannot hold a login for long.
- At most two requests run at the same time. A third request gets an error at once, which protects the computer from
  overload.
- After three failed requests in a row, a circuit breaker stops all requests to the model for 30 seconds. A circuit
  breaker works like a fuse: it stops the calls to a broken part, and the logins do not wait for it. A wrong answer does
  not count as a failure, because the model works and only answers badly.
- The service reads at most 64 KiB from an answer, which limits what a broken server can send.
- At start-up, the service checks the digest (a fingerprint) of the model and does not start with another one. Nobody
  can replace the model without notice.

The service also does not start without Ollama. In this project, the model is a normal part of the checks. A missing
model is therefore an error that the operator sees at once, not a quiet loss of protection. The model could be made
optional, because every failure path already ends with the decision of the rules.

## The Test

The program `cmd/evaluate` creates 620 synthetic logins (made by a program, not taken from real users). It has 300
logins from a normal mix and 40 logins from each of eight hard scenarios, for example a traveler, a mistyped code, a new
phone, slow credential stuffing and a SIM swap. Each scenario sets the signals and the label (attack or real owner). The
program knows the right answer for every login.

The "share of attacks found" is the part of the attacks that the service limited or refused. The "false positive rate"
is the part of the real owners that the service limited by mistake.

| Part of the data | Who decides | Share of attacks found | False positive rate |
| --- | --- | --- | --- |
| Normal mix | rules | 0.855 | 0.000 |
| Normal mix | rules and model | 0.855 | 0.000 |
| Hard scenarios | rules | 0.725 | 0.713 |
| Hard scenarios | rules and model | 0.725 | 0.713 |

The false positive rate of 0.000 in the normal mix is not a surprise, because this part has mostly simple cases. The
hard scenarios show where the rules are weak. For example, the rules limit all real owners who mistype a code or use a
new phone.

The model made the decision stricter for none of the 620 logins, and the rows with and without the model are the same.
Its step took 2.18 seconds for half of the requests and 3.35 seconds or less for 95 percent of them. Four requests
failed.

These facts are certain: the controls changed no decision, a failed request changed no decision, and the model step
costs about two seconds. The test cannot show that the model is useful. The labels come from the same signals that the
rules use, and the other values that the model sees are random or come from the signals. The data therefore has nothing
that the model could find and the rules could not, and a result of zero was almost certain. A real answer needs real
attack data, and the project does not have it. The data also has no VPN logins and says nothing about the VPN signal.

The numbers come from one run on one computer, made before the VPN signal was added and before the service used the
LangChain library. With that library, the check refused about one answer in six, because the model wrote the login
method as a reason, and this is not in the fixed list. One sentence in the prompt reduced this to 3 refused answers in a
sample of 130 logins. A refused answer counts as no advice, and the model made no login stricter in any run. The full
run was not repeated.

## What Can Be Reused

1. Give the model no more power than "make it stricter", and let every error end in the decision of the rules. Then a
   failure of the model cannot weaken the login.
2. Build the prompt only from fields with a fixed format. Then no user can put instructions into it.
3. Check the answer strictly before the program uses it. A JSON mode is not a check, because the model can still break
   the format.
4. Limit the time, the parallel requests and the size of the answer. Count failures for the circuit breaker, but not
   wrong answers.
5. Say what the test cannot show. The controls can be measured. The value of the model needs data that the project does
   not have.
