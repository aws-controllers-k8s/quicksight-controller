	if !equality.Semantic.DeepEqual(a.ko.Status.VersionNumber, b.ko.Status.VersionNumber) {
		delta.Add("Status.VersionNumber", a.ko.Status.VersionNumber, b.ko.Status.VersionNumber)
	}

