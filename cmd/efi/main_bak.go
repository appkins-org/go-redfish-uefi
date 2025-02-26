package main

// func main() {
// 	// Define command-line flags
// 	var (
// 		listFlag     = flag.Bool("list", false, "List boot entries")
// 		orderFlag    = flag.Bool("order", false, "Show boot order")
// 		getFlag      = flag.String("get", "", "Get boot entry by ID (hex)")
// 		activeFlag   = flag.Bool("active", false, "Set boot entry active")
// 		inactiveFlag = flag.Bool("inactive", false, "Set boot entry inactive")
// 		deleteFlag   = flag.String("delete", "", "Delete boot entry by ID (hex)")
// 		sysfsFlag    = flag.String("sysfs", "/sys/firmware/efi/efivars", "Path to sysfs efi variables")
// 		helpFlag     = flag.Bool("help", false, "Show help")
// 	)

// 	flag.Parse()

// 	if *helpFlag {
// 		flag.Usage()
// 		os.Exit(0)
// 	}

// 	// Create variable store
// 	store := efi.NewVariableStore(*sysfsFlag)

// 	// Process commands
// 	if *listFlag {
// 		listBootEntries(store)
// 	} else if *orderFlag {
// 		showBootOrder(store)
// 	} else if *getFlag != "" {
// 		id, err := strconv.ParseUint(*getFlag, 16, 16)
// 		if err != nil {
// 			log.Fatalf("Invalid boot entry ID: %v", err)
// 		}
// 		getBootEntry(store, uint16(id))
// 	} else if *activeFlag || *inactiveFlag {
// 		if flag.NArg() < 1 {
// 			log.Fatalf("Boot entry ID required")
// 		}
// 		id, err := strconv.ParseUint(flag.Arg(0), 16, 16)
// 		if err != nil {
// 			log.Fatalf("Invalid boot entry ID: %v", err)
// 		}
// 		setBootEntryActive(store, uint16(id), *activeFlag)
// 	} else if *deleteFlag != "" {
// 		id, err := strconv.ParseUint(*deleteFlag, 16, 16)
// 		if err != nil {
// 			log.Fatalf("Invalid boot entry ID: %v", err)
// 		}
// 		deleteBootEntry(store, uint16(id))
// 	} else {
// 		flag.Usage()
// 	}
// }

// func listBootEntries(store *efi.VariableStore) {
// 	entries, err := store.GetOrderedBootEntries()
// 	if err != nil {
// 		log.Fatalf("Failed to list boot entries: %v", err)
// 	}

// 	fmt.Println("Boot Entries:")
// 	for i, entry := range entries {
// 		activeStr := " "
// 		if entry.GetActiveStatus() {
// 			activeStr = "*"
// 		}
// 		fmt.Printf("%d) %s %s\n", i+1, activeStr, entry.String())
// 	}
// }

// func showBootOrder(store *efi.VariableStore) {
// 	order, err := store.GetBootOrder()
// 	if err != nil {
// 		log.Fatalf("Failed to get boot order: %v", err)
// 	}

// 	fmt.Println("Boot Order:")
// 	for i, id := range order {
// 		fmt.Printf("%d) Boot%04X\n", i+1, id)
// 	}
// }

// func getBootEntry(store *efi.VariableStore, id uint16) {
// 	entry, err := store.GetBootEntry(id)
// 	if err != nil {
// 		log.Fatalf("Failed to get boot entry: %v", err)
// 	}

// 	fmt.Printf("Boot%04X: %s\n", id, entry.String())
// }

// func setBootEntryActive(store *efi.VariableStore, id uint16, active bool) {
// 	entry, err := store.GetBootEntry(id)
// 	if err != nil {
// 		log.Fatalf("Failed to get boot entry: %v", err)
// 	}

// 	entry.SetActiveStatus(active)

// 	err = store.SetBootEntry(id, entry)
// 	if err != nil {
// 		log.Fatalf("Failed to update boot entry: %v", err)
// 	}

// 	fmt.Printf("Boot%04X %s\n", id,
// 		map[bool]string{true: "activated", false: "deactivated"}[active])
// }

// func deleteBootEntry(store *efi.VariableStore, id uint16) {
// 	err := store.DeleteBootEntry(id)
// 	if err != nil {
// 		log.Fatalf("Failed to delete boot entry: %v", err)
// 	}

// 	fmt.Printf("Boot%04X deleted\n", id)

// 	// Update boot order
// 	order, err := store.GetBootOrder()
// 	if err != nil {
// 		log.Printf("Warning: Failed to get boot order: %v", err)
// 		return
// 	}

// 	// Remove ID from boot order
// 	newOrder := make([]uint16, 0, len(order))
// 	for _, entryID := range order {
// 		if entryID != id {
// 			newOrder = append(newOrder, entryID)
// 		}
// 	}

// 	// Update boot order
// 	if len(newOrder) != len(order) {
// 		err = store.SetBootOrder(newOrder)
// 		if err != nil {
// 			log.Printf("Warning: Failed to update boot order: %v", err)
// 		}
// 	}
// }
