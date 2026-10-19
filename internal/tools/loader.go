package tools

// func LoadRegistry(ctx context.Context, repository Repository, integrationID uuid.UUID) (*Registry, error) {
// 	if repository == nil {
// 		return nil, fmt.Errorf("tool repository is required")
// 	}
// 	if integrationID == "" {
// 		return nil, fmt.Errorf("integration ID is required")
// 	}

// 	records, err := repository.ListEnabled(ctx, integrationID)
// 	if err != nil {
// 		return nil, fmt.Errorf("load tools for integration %q: %w", integrationID, err)
// 	}

// 	registry := NewRegistry()

// 	for _, record := range records {
// 		if record.IntegrationID != integrationID {
// 			return nil, fmt.Errorf(
// 				"repository returned tool %q for unexpected integration",
// 				record.ID,
// 			)
// 		}
// 		if !record.Enabled || record.DeletedAt != nil {
// 			continue
// 		}

// 		if err := registry.Register(NewRegisteredTool(record)); err != nil {
// 			return nil, fmt.Errorf(
// 				"register database tool %q: %w",
// 				record.Definition.Name,
// 				err,
// 			)
// 		}
// 	}

// 	return registry, nil
// }
