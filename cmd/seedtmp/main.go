package main

// Seeds a realistic ShiftMaster scenario for verifying the assistant end to
// end through the Home page: one department, an overnight shift, a few people
// with known passwords, leave types with balances, tasks, service plans, and a
// knowledge document.

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dsn := os.Getenv("DSN")
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	ctx := context.Background()
	must := func(err error, what string) {
		if err != nil {
			log.Fatalf("%s: %v", what, err)
		}
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("Passw0rd!2026"), 10)

	var deptOps, deptSup string
	must(pool.QueryRow(ctx, `INSERT INTO departments (name, department_code) VALUES ('Operations','OPS')
		ON CONFLICT (department_code) DO UPDATE SET name=EXCLUDED.name RETURNING id`).Scan(&deptOps), "dept ops")
	must(pool.QueryRow(ctx, `INSERT INTO departments (name, department_code) VALUES ('Support','SUP')
		ON CONFLICT (department_code) DO UPDATE SET name=EXCLUDED.name RETURNING id`).Scan(&deptSup), "dept sup")

	var nightShift, dayShift string
	must(pool.QueryRow(ctx, `INSERT INTO shifts (name, shift_code, start_time, end_time, department_id)
		VALUES ('Evening','EVE','16:30','00:30',$1) ON CONFLICT (shift_code) DO UPDATE SET start_time=EXCLUDED.start_time RETURNING id`, deptOps).Scan(&nightShift), "night shift")
	must(pool.QueryRow(ctx, `INSERT INTO shifts (name, shift_code, start_time, end_time, department_id)
		VALUES ('Morning','MRN','08:00','16:00',$1) ON CONFLICT (shift_code) DO UPDATE SET start_time=EXCLUDED.start_time RETURNING id`, deptOps).Scan(&dayShift), "day shift")

	person := func(code, first, last, email, role, dept, shift string) string {
		var id string
		must(pool.QueryRow(ctx, `INSERT INTO employees (employee_code, first_name, last_name, gender, email, password_hash,
			hire_date, role, department_id, default_shift_id, weekly_off_days, status)
			VALUES ($1,$2,$3,'male',$4,$5,CURRENT_DATE - INTERVAL '2 years',$6,$7,$8,1,'active')
			ON CONFLICT (employee_code) DO UPDATE SET password_hash=EXCLUDED.password_hash, role=EXCLUDED.role,
			  department_id=EXCLUDED.department_id, default_shift_id=EXCLUDED.default_shift_id, status='active'
			RETURNING id`, code, first, last, email, string(hash), role, dept, shift).Scan(&id), "employee "+code)
		return id
	}
	admin := person("ADM001", "Haider", "Admin", "admin@fiberx.iq", "admin", deptOps, dayShift)
	mgr := person("MGR001", "Sara", "Manager", "manager@fiberx.iq", "manager", deptOps, dayShift)
	lead := person("TL001", "Omar", "Lead", "lead@fiberx.iq", "team_leader", deptOps, nightShift)
	emp := person("EMP001", "Ali", "Hassan", "ali@fiberx.iq", "employee", deptOps, nightShift)
	emp2 := person("EMP002", "Noor", "Kareem", "noor@fiberx.iq", "employee", deptOps, nightShift)
	_ = admin

	_, err = pool.Exec(ctx, `INSERT INTO department_managers (department_id, manager_id) VALUES ($1,$2),($3,$2)
		ON CONFLICT DO NOTHING`, deptOps, mgr, deptSup)
	if err != nil {
		log.Printf("department_managers: %v", err)
	}

	// Leave types and balances.
	var hourly, annual string
	_, _ = pool.Exec(ctx, `DELETE FROM employee_leave_balances`)
	_, _ = pool.Exec(ctx, `DELETE FROM leave_types WHERE name_en IN ('Hourly leave','Annual leave')`)
	must(pool.QueryRow(ctx, `INSERT INTO leave_types (name_ar, name_en, unit, reset_cycle, is_active, is_hourly, days_per_year)
		VALUES ('زمنية','Hourly leave','hours','monthly',true,true,0) RETURNING id`).Scan(&hourly), "hourly type")
	must(pool.QueryRow(ctx, `INSERT INTO leave_types (name_ar, name_en, unit, reset_cycle, is_active, is_hourly, days_per_year)
		VALUES ('سنوية','Annual leave','days','annual',true,false,21) RETURNING id`).Scan(&annual), "annual type")

	year := time.Now().Year()
	for _, e := range []string{emp, emp2, lead, mgr} {
		for _, pair := range []struct {
			t string
			a float64
			m int
		}{{hourly, 12, 0}, {annual, 21, 0}} {
			_, err := pool.Exec(ctx, `INSERT INTO employee_leave_balances (employee_id, leave_type_id, year, month, allocated_amount, used_amount)
				VALUES ($1,$2,$3,$4,$5,0)
				ON CONFLICT (employee_id, leave_type_id, year, month) DO UPDATE SET allocated_amount=EXCLUDED.allocated_amount`,
				e, pair.t, year, pair.m, pair.a)
			if err != nil {
				log.Printf("balance: %v", err)
			}
		}
	}

	// Service catalogue: a province, a category and three plans.
	var province, category string
	if pool.QueryRow(ctx, `SELECT id FROM provinces WHERE name='Baghdad' AND department_id=$1`, deptOps).Scan(&province) != nil {
		must(pool.QueryRow(ctx, `INSERT INTO provinces (name, department_id) VALUES ('Baghdad', $1) RETURNING id`, deptOps).Scan(&province), "province")
	}
	if pool.QueryRow(ctx, `SELECT id FROM service_categories WHERE province_id=$1 AND name='FTTH Home'`, province).Scan(&category) != nil {
		must(pool.QueryRow(ctx, `INSERT INTO service_categories (province_id, name, sort_order, created_by)
			VALUES ($1,'FTTH Home',1,$2) RETURNING id`, province, mgr).Scan(&category), "category")
	}
	_, _ = pool.Exec(ctx, `DELETE FROM service_plans WHERE category_id=$1`, category)
	for _, p := range []struct {
		name  string
		price float64
		speed string
		days  int
	}{
		{"Basic 25", 25000, "25 Mbps", 30},
		{"Standard 50", 40000, "50 Mbps", 30},
		{"Premium 100", 65000, "100 Mbps", 30},
	} {
		_, err := pool.Exec(ctx, `INSERT INTO service_plans (category_id, name, price, duration_days, speed, connection_type, router_included, is_active, sort_order, created_by)
			VALUES ($1,$2,$3,$4,$5,'FTTH',true,true,1,$6)`, category, p.name, p.price, p.days, p.speed, mgr)
		if err != nil {
			log.Printf("plan %s: %v", p.name, err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO province_department_shares (province_id, department_id, granted_by) VALUES ($1,$2,$4),($1,$3,$4)`, province, deptOps, deptSup, mgr); err != nil {
		log.Printf("province shares: %v", err)
	}

	// A knowledge document the assistant should find and cite.
	_, err = pool.Exec(ctx, `INSERT INTO help_documents (department_id, title, content, created_by)
		VALUES ($1,'سياسة الإجازات والتأخير',
		 'طلب الإجازة يقدّم قبل 48 ساعة على الأقل من موعدها. إذا كان دوامك ليلي وتريد إجازة زمنية، قدّم الطلب قبل بداية الشفت وستحتاج موافقة قائد الفريق ثم المدير. التأخير أكثر من 15 دقيقة يُسجّل ويُخصم من الرصيد الزمني.',
		 $2)`, deptOps, mgr)
	if err != nil {
		log.Printf("help doc: %v", err)
	}

	// A published week of schedule for the whole department, so the assistant
	// has real shifts to resolve — including the overnight one the temporal
	// rules exist for.
	var week string
	_, _ = pool.Exec(ctx, `DELETE FROM employee_shifts`)
	_, _ = pool.Exec(ctx, `DELETE FROM weekly_schedule`)
	_, _ = pool.Exec(ctx, `DELETE FROM help_documents`)
	must(pool.QueryRow(ctx, `INSERT INTO weekly_schedule (week_start_date, week_end_date, status, department_id, created_by, published_by, published_at)
		VALUES (CURRENT_DATE - INTERVAL '3 days', CURRENT_DATE + INTERVAL '10 days', 'published', $1, $2, $2, now())
		RETURNING id`, deptOps, mgr).Scan(&week), "weekly schedule")

	for _, assignment := range []struct {
		emp   string
		shift string
	}{{emp, nightShift}, {emp2, nightShift}, {lead, nightShift}, {mgr, dayShift}} {
		for day := -3; day <= 10; day++ {
			status := "working"
			// One day off a week, so "am I off tomorrow" has a real answer.
			if (day+14)%7 == 5 {
				status = "off"
			}
			if _, err := pool.Exec(ctx,
				`INSERT INTO employee_shifts (schedule_id, employee_id, shift_id, shift_date, shift_status)
				 VALUES ($1,$2,$3, CURRENT_DATE + make_interval(days => $4), $5::shift_status_type)`,
				week, assignment.emp, assignment.shift, day, status); err != nil {
				log.Printf("employee_shift day %d: %v", day, err)
				break
			}
		}
	}

	fmt.Println("seeded.")
	fmt.Println("  admin@fiberx.iq / manager@fiberx.iq / lead@fiberx.iq / ali@fiberx.iq / noor@fiberx.iq")
	fmt.Println("  password: Passw0rd!2026")
	fmt.Printf("  ops department %s, evening shift %s (16:30-00:30)\n", deptOps, nightShift)
}
