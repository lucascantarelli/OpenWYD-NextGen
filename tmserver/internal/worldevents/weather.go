package worldevents

// Valores de clima. O legado só grava 0/1/2 em CurrentWeather
// (Server.cpp:688); como cada um renderiza no cliente é UNVERIFIED, então
// eles continuam numerados. O efeito de gameplay é conhecido: BASE_GetSkillDamage
// reduz InstanceType 2 e aumenta 5 sob clima 1, e aumenta InstanceType 3
// sob clima 2 (Basedef.cpp:6998 → combat.SkillBaseDamage).
const (
	WeatherClear = int32(0)
	WeatherOne   = int32(1)
	WeatherTwo   = int32(2)
)

// weatherRollMod é o `rand() % 1200` legado (ProcessSecMinTimer.cpp:2791).
const weatherRollMod = 1200

// weatherBands é a tabela de sorteio legada com os ramos REORDENADOS.
//
// O original é uma cadeia if/else-if que testa [0,260)→0 primeiro
// (ProcessSecMinTimer.cpp:2793-2810). Como [30,50) e [55,60) são subfaixas
// estritas de [0,260), o primeiro ramo sempre vence e os climas 1 e 2 ficam
// inalcançáveis — na prática o clima nunca muda sozinho, só pela
// sobrescrita ForceWeather de GM. Decisão da issue #116: CORRIGIR a ordem em vez
// de preservar o bug, para o cliente realmente ver o clima mudar. Toda constante
// de borda abaixo é a do legado, intocada; só a ordem dos testes difere.
//
// No espaço completo de 1200 valores: 0→235 (a faixa de 260 menos os 25 valores
// que as faixas estreitas agora tomam), 1→20, 2→5, e 940 valores não escolhem faixa.
var weatherBands = [...]struct {
	lo, hi  int // semiaberto [lo, hi)
	weather int32
}{
	{30, 50, WeatherOne},
	{55, 60, WeatherTwo},
	{0, 260, WeatherClear},
}

// RollWeather faz o sorteio de clima por minuto e reporta o próximo valor.
//
// Consome EXATAMENTE um sorteio por chamada, e quem chama precisa chamá-lo todo minuto
// mesmo enquanto uma sobrescrita de GM fixa o clima: o legado sorteia
// incondicionalmente antes de olhar ForceWeather, então pular o sorteio
// dessincronizaria o fluxo.
//
// changed é falso quando o sorteio não escolheu faixa, ou escolheu a faixa que
// já está em vigor (a guarda `CurrentWeather != N` do legado) — nos dois casos
// quem chama não pode transmitir.
func RollWeather(r Rand, current int32) (next int32, changed bool) {
	roll := r.Intn(weatherRollMod)
	for _, b := range weatherBands {
		if roll < b.lo || roll >= b.hi {
			continue
		}
		if b.weather == current {
			return current, false
		}
		return b.weather, true
	}
	return current, false
}

// ValidWeather diz se v é um dos três valores que o legado grava.
// O comando de GM /weather legado não validava nada e mandaria um int
// arbitrário ao cliente (imple.cpp:1122-1129); nós recusamos.
func ValidWeather(v int32) bool {
	return v >= WeatherClear && v <= WeatherTwo
}
