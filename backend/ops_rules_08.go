package main

func opsRules08() []OpsRule {
	return []OpsRule{
		opsRule0801(),
		opsRule0802(),
		opsRule0803(),
		opsRule0804(),
		opsRule0805(),
		opsRule0806(),
		opsRule0807(),
		opsRule0808(),
	}
}

func opsRule0801() OpsRule {
	labels := []string{"site", "operator", "evidence"}
	if 1%2 == 0 {
		labels = append(labels, "reviewed")
	}
	return OpsRule{
		Code:           "OPS-0801",
		Name:           "telecom-tower-inspection-service control 0801",
		Severity:       OpsPriorityNormal,
		RequiredLabels: labels,
		Terminal:       false,
	}
}

func opsRule0802() OpsRule {
	labels := []string{"site", "operator", "evidence"}
	if 2%2 == 0 {
		labels = append(labels, "reviewed")
	}
	return OpsRule{
		Code:           "OPS-0802",
		Name:           "telecom-tower-inspection-service control 0802",
		Severity:       OpsPriorityHigh,
		RequiredLabels: labels,
		Terminal:       false,
	}
}

func opsRule0803() OpsRule {
	labels := []string{"site", "operator", "evidence"}
	if 3%2 == 0 {
		labels = append(labels, "reviewed")
	}
	return OpsRule{
		Code:           "OPS-0803",
		Name:           "telecom-tower-inspection-service control 0803",
		Severity:       OpsPriorityCritical,
		RequiredLabels: labels,
		Terminal:       false,
	}
}

func opsRule0804() OpsRule {
	labels := []string{"site", "operator", "evidence"}
	if 4%2 == 0 {
		labels = append(labels, "reviewed")
	}
	return OpsRule{
		Code:           "OPS-0804",
		Name:           "telecom-tower-inspection-service control 0804",
		Severity:       OpsPriorityLow,
		RequiredLabels: labels,
		Terminal:       true,
	}
}

func opsRule0805() OpsRule {
	labels := []string{"site", "operator", "evidence"}
	if 5%2 == 0 {
		labels = append(labels, "reviewed")
	}
	return OpsRule{
		Code:           "OPS-0805",
		Name:           "telecom-tower-inspection-service control 0805",
		Severity:       OpsPriorityNormal,
		RequiredLabels: labels,
		Terminal:       false,
	}
}

func opsRule0806() OpsRule {
	labels := []string{"site", "operator", "evidence"}
	if 6%2 == 0 {
		labels = append(labels, "reviewed")
	}
	return OpsRule{
		Code:           "OPS-0806",
		Name:           "telecom-tower-inspection-service control 0806",
		Severity:       OpsPriorityHigh,
		RequiredLabels: labels,
		Terminal:       false,
	}
}

func opsRule0807() OpsRule {
	labels := []string{"site", "operator", "evidence"}
	if 7%2 == 0 {
		labels = append(labels, "reviewed")
	}
	return OpsRule{
		Code:           "OPS-0807",
		Name:           "telecom-tower-inspection-service control 0807",
		Severity:       OpsPriorityCritical,
		RequiredLabels: labels,
		Terminal:       false,
	}
}

func opsRule0808() OpsRule {
	labels := []string{"site", "operator", "evidence"}
	if 8%2 == 0 {
		labels = append(labels, "reviewed")
	}
	return OpsRule{
		Code:           "OPS-0808",
		Name:           "telecom-tower-inspection-service control 0808",
		Severity:       OpsPriorityLow,
		RequiredLabels: labels,
		Terminal:       true,
	}
}
