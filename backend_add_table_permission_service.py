import re

with open('internal/service/info_table_service.go', 'r') as f:
    svc = f.read()

# 1. Update computeAccessLevel signature
svc = svc.replace(
    'func (s *InfoTableService) computeAccessLevel(ctx context.Context, table *models.InfoTable, reqEmployeeID uuid.UUID, reqRole string, reqDepID *uuid.UUID) string {',
    'func (s *InfoTableService) computeAccessLevel(ctx context.Context, table *models.InfoTable, reqEmployeeID uuid.UUID, reqRole string, reqDepID *uuid.UUID, canCreateTables bool) string {'
)

# 2. Add canCreateTables check in computeAccessLevel
# Right after: if reqRole == "admin" || (table.CreatedBy != nil && *table.CreatedBy == reqEmployeeID) { return "manage" }
if "if canCreateTables {" not in svc:
    svc = svc.replace(
        'if reqRole == "admin" || (table.CreatedBy != nil && *table.CreatedBy == reqEmployeeID) {\n\t\treturn "manage"\n\t}',
        'if reqRole == "admin" || (table.CreatedBy != nil && *table.CreatedBy == reqEmployeeID) {\n\t\treturn "manage"\n\t}\n\n\t// If employee has global permission to create tables, grant manage access\n\tif canCreateTables {\n\t\treturn "manage"\n\t}'
    )

# 3. Update callers of computeAccessLevel
# In GetVisibleTables:
# We need to fetch emp to get canCreateTables
get_visible_replacement = """	tables, err := s.repo.GetVisibleTables(ctx, employeeID, role, departmentID)
	if err != nil {
		return nil, err
	}
	emp, _ := s.employeeRepo.GetByID(ctx, employeeID)
	var filteredTables []models.InfoTable
	for i := range tables {
		accessLevel := s.computeAccessLevel(ctx, &tables[i], employeeID, role, departmentID, emp != nil && emp.CanCreateTables)
"""
svc = re.sub(r'tables, err := s\.repo\.GetVisibleTables.*?for i := range tables \{\n\s+accessLevel := s\.computeAccessLevel\(ctx, &tables\[i\], employeeID, role, departmentID\)', get_visible_replacement, svc, flags=re.DOTALL)

# In GetTableByID:
# Needs to fetch emp.
get_by_id_replacement = """	table, err := s.repo.GetTableByID(ctx, tableID)
	if err != nil {
		return nil, err
	}

	emp, _ := s.employeeRepo.GetByID(ctx, reqEmployeeID)
	canCreate := emp != nil && emp.CanCreateTables

	if reqRole == "admin" || (table.CreatedBy != nil && *table.CreatedBy == reqEmployeeID) {
		table.MyAccessLevel = s.computeAccessLevel(ctx, table, reqEmployeeID, reqRole, reqDepID, canCreate)
		return table, nil
	}

	// Check explicit employee accesses first for blocking
	empAccesses, err := s.repo.GetEmployeeAccesses(ctx, tableID)
	if err == nil {
		for _, ea := range empAccesses {
			if ea.EmployeeID == reqEmployeeID && ea.AccessLevel == "none" {
				return nil, errors.New("unauthorized access to table (explicitly blocked)")
			}
		}
	}

	// Check department matching
	if reqDepID != nil && table.DepartmentID != nil && *reqDepID == *table.DepartmentID {
		table.MyAccessLevel = s.computeAccessLevel(ctx, table, reqEmployeeID, reqRole, reqDepID, canCreate)
		return table, nil
	}

	// Check department accesses
	depAccesses, err := s.repo.GetDepartmentAccesses(ctx, tableID)
	if err == nil {
		for _, da := range depAccesses {
			if reqDepID != nil && da.DepartmentID == *reqDepID {
				table.MyAccessLevel = s.computeAccessLevel(ctx, table, reqEmployeeID, reqRole, reqDepID, canCreate)
				return table, nil
			}
		}
	}

	// Check employee accesses (again for granting)
	if err == nil {
		for _, ea := range empAccesses {
			if ea.EmployeeID == reqEmployeeID {
				table.MyAccessLevel = s.computeAccessLevel(ctx, table, reqEmployeeID, reqRole, reqDepID, canCreate)
				return table, nil
			}
		}
	}

	return nil, errors.New("unauthorized access to table")"""
svc = re.sub(r'table, err := s\.repo\.GetTableByID\(ctx, tableID\)\n\s+if err != nil \{\n\s+return nil, err\n\s+\}.*?return nil, errors\.New\("unauthorized access to table"\)', get_by_id_replacement, svc, flags=re.DOTALL)

# 4. Update HasWriteAccess and HasManageAccessRight
has_write_replacement = """func (s *InfoTableService) HasWriteAccess(ctx context.Context, tableID uuid.UUID, reqEmployeeID uuid.UUID, reqRole string, reqDepID *uuid.UUID) bool {
	if reqRole == "admin" {
		return true
	}
	emp, _ := s.employeeRepo.GetByID(ctx, reqEmployeeID)
	if emp != nil && emp.CanCreateTables {
		return true
	}
"""
svc = re.sub(r'func \(s \*InfoTableService\) HasWriteAccess.*?if reqRole == "admin" \{\n\s+return true\n\s+\}', has_write_replacement, svc, flags=re.DOTALL)

has_manage_replacement = """func (s *InfoTableService) HasManageAccessRight(ctx context.Context, tableID uuid.UUID, reqEmployeeID uuid.UUID, reqRole string, reqDepID *uuid.UUID) bool {
	if reqRole == "admin" {
		return true
	}
	emp, _ := s.employeeRepo.GetByID(ctx, reqEmployeeID)
	if emp != nil && emp.CanCreateTables {
		return true
	}
"""
svc = re.sub(r'func \(s \*InfoTableService\) HasManageAccessRight.*?if reqRole == "admin" \{\n\s+return true\n\s+\}', has_manage_replacement, svc, flags=re.DOTALL)

with open('internal/service/info_table_service.go', 'w') as f:
    f.write(svc)

