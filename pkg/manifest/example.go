package manifest

func Example(exp int64) *Manifest {

	M := &Manifest{}

	M.UISets = make(map[Ref]*UISet)
	M.UISets["penduino"] = &UISet{"penduino-basic-ui-v1.0", "penduino-advanced-ui-v1.0"}
	M.UISets["spinner"] = &UISet{"spinner-basic-ui-v1.0", "spinner-advanced-ui-v1.0"}

	M.Descriptions = make(map[Ref]*Description)

	M.Descriptions["penduino-activity-v1.0"] = &Description{
		Name:    "Penduino",
		Type:    "penduino-activity-v1.0",
		Short:   "Electromagnetic Pendulum",
		Long:    `A simple pendulum with electromagnetic drive system producing simple harmonic motion. The drive and braking effect are variable. They are determined by how much of the pendulum's travel the coil remains energised for. The pendulum can also be slowed down by short-circuiting the coil, without applying any power, or left to swing freely with no drive or braking.`,
		Further: "https://static.practable.io/info/penduino-v1.0",
		Thumb:   "https://assets.practable.io/images/penduino-v1.0/thumb.png",
		Image:   "https://assets.practable.io/images/penduino-v1.0/image.png",
	}

	M.Descriptions["spinner-activity-v1.0"] = &Description{
		Name:    "Spinner",
		Type:    "spinner-activity-v1.0",
		Short:   "Weighted spinning disk",
		Long:    ` A simple weighted spinning disk driven by a brushed DC motor. The spin speed can be set by the duty cycle of the drive signal, either directly, or by using a proportional-integral-derivative (PID) control loop. The disk position can also be set by using a PID loop.`,
		Further: "https://static.practable.io/info/spinner-v1.0",
		Thumb:   "https://assets.practable.io/images/spinner-v1.0/thumb.png",
		Image:   "https://assets.practable.io/images/spinner-v1.0/image.png",
	}

	M.UIs = make(map[Ref]*UI)

	M.UIs["penduino-basic-ui-v1.0"] = &UI{
		Description: Description{
			Name:    "Penduino (Basic)",
			Type:    "penduino-basic-ui-v1.0",
			Short:   "Control an Electromagnetic Pendulum",
			Long:    `Start the pendulum swinging, and observe the difference in decay time for three different types of braking - no braking, short circuit load, and energised coil.`,
			Further: "https://static.practable.io/info/penduino-basic-ui-v1.0",
			Thumb:   "https://assets.practable.io/images/penduino-basic-ui-v1.0/thumb.png",
			Image:   "https://assets.practable.io/images/penduino-basic-ui-v1.0/image.png",
		},
		URL:             "https://static.practable.io/ui/penduino-basic-ui-v1.0?data={{data}}&video={{video}}",
		StreamsRequired: []string{"data", "video"},
	}

	M.UIs["penduino-advanced-ui-v1.0"] = &UI{
		Description: Description{
			Name:    "Penduino (Advanced)",
			Type:    "penduino-advanced-ui-v1.0",
			Short:   "Control an Electromagnetic Pendulum",
			Long:    `Start the pendulum swinging, and observe the difference in decay time for three different types of braking - no braking, short circuit load, and energised coil. Explore the impact of different settings for the drive and brake.`,
			Further: "https://static.practable.io/info/penduino-advanced-ui-v1.0",
			Thumb:   "https://assets.practable.io/images/penduino-advanced-ui-v1.0/thumb.png",
			Image:   "https://assets.practable.io/images/penduino-advanced-ui-v1.0/image.png",
		},
		URL:             "https://static.practable.io/ui/penduino-advanced-ui-v1.0?data={{data}}&video={{video}}",
		StreamsRequired: []string{"data", "video"},
	}

	M.UIs["spinner-basic-ui-v1.0"] = &UI{
		Description: Description{
			Name:    "Spinner (Basic)",
			Type:    "spinner-basic-ui-v1.0",
			Short:   "Control a spinning weighted disk",
			Long:    `Set the drive percentage and observe the steady state speed.`,
			Further: "https://static.practable.io/info/spinner-basic-ui-v1.0",
			Thumb:   "https://assets.practable.io/images/spinner-basic-ui-v1.0/thumb.png",
			Image:   "https://assets.practable.io/images/spinner-basic-ui-v1.0/image.png",
		},
		URL:             "https://static.practable.io/ui/spinner-basic-ui-v1.0?data={{data}}&video={{video}}",
		StreamsRequired: []string{"data", "video"},
	}

	M.UIs["spinner-advanced-ui-v1.0"] = &UI{
		Description: Description{
			Name:    "Spinner (Advanced)",
			Type:    "spinner-advanced-ui-v1.0",
			Short:   "Control a spinning weighted disk",
			Long:    `Use a PID loop to achieve a specificed spin speed`,
			Further: "https://static.practable.io/info/spinner-advanced-ui-v1.0",
			Thumb:   "https://assets.practable.io/images/spinner-advanced-ui-v1.0/thumb.png",
			Image:   "https://assets.practable.io/images/spinner-advanced-ui-v1.0/image.png",
		},
		URL:             "https://static.practable.io/ui/spinner-advanced-ui-v1.0?data={{data}}&video={{video}}",
		StreamsRequired: []string{"data", "video"},
	}

	M.Activities = make(map[Ref]*Activity)

	spa0 := make(map[string]*Stream)
	spa0["data"] = &Stream{
		For:            "data",
		URL:            "https://relay-access.practable.io/session/penduino-activity-00-data",
		Audience:       "https://relay-access.practable.io",
		ConnectionType: "session",
		Topic:          "penduino-activity-00-data",
		Verb:           "POST",
		Scopes:         []string{"read"},
	}
	spa0["video"] = &Stream{
		For:            "data",
		URL:            "https://relay-access.practable.io/session/penduino-activity-00-video",
		Audience:       "https://relay-access.practable.io",
		ConnectionType: "session",
		Topic:          "penduino-activity-00-video",
		Verb:           "POST",
		Scopes:         []string{"read", "write"},
	}

	cfg0 := Config{URL: "https://assets.practable.io/config/experiments/penduino/penduino00-0.0.json"}

	M.Activities["penduino-activity-00"] = &Activity{
		Config:      cfg0,
		Description: "penduino-activity-v1.0",
		UISet:       "penduino",
		ExpiresAt:   exp,
		Streams:     spa0,
	}

	spa1 := make(map[string]*Stream)
	spa1["data"] = &Stream{
		For:            "data",
		URL:            "https://relay-access.practable.io/session/penduino-activity-01-data",
		Audience:       "https://relay-access.practable.io",
		ConnectionType: "session",
		Topic:          "penduino-activity-01-data",
		Verb:           "POST",
		Scopes:         []string{"read"},
	}
	spa1["video"] = &Stream{
		For:            "data",
		URL:            "https://relay-access.practable.io/session/penduino-activity-01-video",
		Audience:       "https://relay-access.practable.io",
		ConnectionType: "session",
		Topic:          "penduino-activity-01-video",
		Verb:           "POST",
		Scopes:         []string{"read", "write"},
	}

	cfg1 := Config{URL: "https://assets.practable.io/config/experiments/penduino/penduino01-0.0.json"}

	M.Activities["penduino-activity-01"] = &Activity{
		Config:      cfg1,
		Description: "penduino-activity-v1.0",
		UISet:       "penduino",
		ExpiresAt:   exp,
		Streams:     spa1,
	}

	spa2 := make(map[string]*Stream)
	spa2["data"] = &Stream{
		For:            "data",
		URL:            "https://relay-access.practable.io/session/penduino-activity-02-data",
		Audience:       "https://relay-access.practable.io",
		ConnectionType: "session",
		Topic:          "penduino-activity-02-data",
		Verb:           "POST",
		Scopes:         []string{"read"},
	}
	spa2["video"] = &Stream{
		For:            "data",
		URL:            "https://relay-access.practable.io/session/penduino-activity-02-video",
		Audience:       "https://relay-access.practable.io",
		ConnectionType: "session",
		Topic:          "penduino-activity-02-video",
		Verb:           "POST",
		Scopes:         []string{"read", "write"},
	}

	cfg2 := Config{URL: "https://assets.practable.io/config/experiments/penduino/penduino02-0.0.json"}

	M.Activities["penduino-activity-02"] = &Activity{
		Config:      cfg2,
		Description: "penduino-activity-v1.0",
		UISet:       "penduino",
		ExpiresAt:   exp,
		Streams:     spa2,
	}

	M.Pools = make(map[Ref]*Pool)

	M.Pools["penduino-everyone"] = &Pool{
		Description: Description{
			Name:    "Penduino (Everyone)",
			Type:    "penduino-pool-v1.0",
			Short:   "Electromagnetic Pendulums",
			Long:    `Explore the operation of simple pendulums using an electromagnetic drive system.`,
			Further: "https://static.practable.io/info/penduino-v1.0",
			Thumb:   "https://assets.practable.io/images/penduino-v1.0/thumb.png",
			Image:   "https://assets.practable.io/images/penduino-v1.0/image.png",
		},
		MinSession: 600,
		MaxSession: 3600,
		Activities: []Ref{"penduino-activity-00", "penduino-activity-01", "penduino-activity-02"},
	}

	ssa0 := make(map[string]*Stream)
	ssa0["data"] = &Stream{
		For:            "data",
		URL:            "https://relay-access.practable.io/session/spinner-activity-00-data",
		Audience:       "https://relay-access.practable.io",
		ConnectionType: "session",
		Topic:          "spinner-activity-00-data",
		Verb:           "POST",
		Scopes:         []string{"read"},
	}
	ssa0["video"] = &Stream{
		For:            "data",
		URL:            "https://relay-access.practable.io/session/spinner-activity-00-video",
		Audience:       "https://relay-access.practable.io",
		ConnectionType: "session",
		Topic:          "spinner-activity-00-video",
		Verb:           "POST",
		Scopes:         []string{"read", "write"},
	}

	M.Activities["spinner-activity-00"] = &Activity{
		Description: "spinner-activity-v1.0",
		UISet:       "spinner",
		ExpiresAt:   exp,
		Streams:     ssa0,
	}

	ssa1 := make(map[string]*Stream)
	ssa1["data"] = &Stream{
		For:            "data",
		URL:            "https://relay-access.practable.io/session/spinner-activity-01-data",
		Audience:       "https://relay-access.practable.io",
		ConnectionType: "session",
		Topic:          "spinner-activity-01-data",
		Verb:           "POST",
		Scopes:         []string{"read"},
	}
	ssa1["video"] = &Stream{
		For:            "data",
		URL:            "https://relay-access.practable.io/session/spinner-activity-01-video",
		Audience:       "https://relay-access.practable.io",
		ConnectionType: "session",
		Topic:          "spinner-activity-01-video",
		Verb:           "POST",
		Scopes:         []string{"read", "write"},
	}

	M.Activities["spinner-activity-01"] = &Activity{
		Description: "spinner-activity-v1.0",
		UISet:       "spinner",
		ExpiresAt:   exp,
		Streams:     ssa1,
	}

	ssa2 := make(map[string]*Stream)
	ssa2["data"] = &Stream{
		For:            "data",
		URL:            "https://relay-access.practable.io/session/spinner-activity-02-data",
		Audience:       "https://relay-access.practable.io",
		ConnectionType: "session",
		Topic:          "spinner-activity-02-data",
		Verb:           "POST",
		Scopes:         []string{"read"},
	}
	ssa2["video"] = &Stream{
		For:            "data",
		URL:            "https://relay-access.practable.io/session/spinner-activity-02-video",
		Audience:       "https://relay-access.practable.io",
		ConnectionType: "session",
		Topic:          "spinner-activity-02-video",
		Verb:           "POST",
		Scopes:         []string{"read", "write"},
	}

	M.Activities["spinner-activity-02"] = &Activity{
		Description: "spinner-activity-v1.0",
		UISet:       "spinner",
		ExpiresAt:   exp,
		Streams:     ssa2,
	}

	M.Pools["spinner-everyone"] = &Pool{
		Description: Description{
			Name:    "Spinner (Everyone)",
			Type:    "spinner-pool-v1.0",
			Short:   "Electromagnetic Pendulums",
			Long:    `Explore the operation of simple pendulums using an electromagnetic drive system.`,
			Further: "https://static.practable.io/info/spinner-v1.0",
			Thumb:   "https://assets.practable.io/images/spinner-v1.0/thumb.png",
			Image:   "https://assets.practable.io/images/spinner-v1.0/image.png",
		},
		MinSession: 600,
		MaxSession: 3600,
		Activities: []Ref{"spinner-activity-00"},
	}

	M.Pools["spinner-controls3"] = &Pool{
		Description: Description{
			Name:    "Spinner (Controls 3)",
			Type:    "spinner-pool-v1.0",
			Short:   "Electromagnetic Pendulums",
			Long:    `Explore the operation of simple pendulums using an electromagnetic drive system.`,
			Further: "https://static.practable.io/info/spinner-v1.0",
			Thumb:   "https://assets.practable.io/images/spinner-v1.0/thumb.png",
			Image:   "https://assets.practable.io/images/spinner-v1.0/image.png",
		},
		MinSession: 600,
		MaxSession: 3600,
		Activities: []Ref{"spinner-activity-01", "spinner-activity-02"},
	}

	M.Groups = make(map[Ref]*Group)

	M.Groups["everyone"] = &Group{
		Pools: []Ref{"penduino-everyone", "spinner-everyone"},
	}

	M.Groups["controls3"] = &Group{
		Pools: []Ref{"spinner-controls3"},
	}

	return M

}
