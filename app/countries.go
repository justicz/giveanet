package main

var allowedCountries map[string]bool

func init() {
	allowedCountries = make(map[string]bool)
	codes := []string{ "none","af","al","dz","as","ad","ao","ai","ag","ar","am","aw","au","at","az","bs","bh","bd","bb","by","be","bz","bj","bm","bt","bo","ba","bw","br","io","vg","bn","bg","bf","bi","kh","cm","ca","cv","bq","ky","cf","td","cl","cn","cx","cc","co","km","cd","cg","ck","cr","ci","hr","cu","cw","cy","cz","dk","dj","dm","do","ec","eg","sv","gq","er","ee","et","fk","fo","fj","fi","fr","gf","pf","ga","gm","ge","de","gh","gi","gr","gl","gd","gp","gu","gt","gg","gn","gw","gy","ht","hn","hk","hu","is","in","id","ir","iq","ie","im","il","it","jm","jp","je","jo","kz","ke","ki","xk","kw","kg","la","lv","lb","ls","lr","ly","li","lt","lu","mo","mk","mg","mw","my","mv","ml","mt","mh","mq","mr","mu","yt","mx","fm","md","mc","mn","me","ms","ma","mz","mm","na","nr","np","nl","nc","nz","ni","ne","ng","nu","nf","kp","mp","no","om","pk","pw","ps","pa","pg","py","pe","ph","pl","pt","pr","qa","re","ro","ru","rw","bl","sh","kn","lc","mf","pm","vc","ws","sm","st","sa","sn","rs","sc","sl","sg","sx","sk","si","sb","so","za","kr","ss","es","lk","sd","sr","sj","sz","se","ch","sy","tw","tj","tz","th","tl","tg","tk","to","tt","tn","tr","tm","tc","tv","vi","ug","ua","ae","gb","us","uy","uz","vu","va","ve","vn","wf","eh","ye","zm","zw","ax" }
	for _, code := range(codes) {
		allowedCountries[code] = true;
	}
}
