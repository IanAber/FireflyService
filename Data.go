package main

import (
	"database/sql"
	"fmt"
	"github.com/bdwilliams/go-jsonify/jsonify"
	"log"
	"net/http"
	"time"
)

/*
*
GetTimeRange returns the start and end times passed as query parameters.
*/
func GetTimeRange(r *http.Request) (start time.Time, end time.Time, err error) {
	params := r.URL.Query()
	values := params["start"]
	if len(values) != 1 {
		err = fmt.Errorf("Exactly one 'start=' value must be supplied for start time")
		return
	}
	timeVal, err := time.Parse("2006-1-2 15:4", values[0])
	if err != nil {
		return
	} else {
		start = timeVal
	}

	values = params["end"]
	if len(values) != 1 {
		err = fmt.Errorf("Exactly one 'start=' value must be supplied for start time")
		return
	}
	timeVal, err = time.Parse("2006-1-2 15:4", values[0])
	if err != nil {
		return
	} else {
		end = timeVal
	}
	if end.Before(start) {
		err = fmt.Errorf("End Time (%s) must be after Start Time (%s)", end.String(), start.String())
		return
	}
	return
}

func getDatabaseRowsAsJSON(pdb *sql.DB, qry string, args ...any) ([]string, error) {
	if pDB == nil {
		return nil, fmt.Errorf("the database is not connected")
	}
	if incidentRows, err := pdb.Query(qry, args...); err != nil {
		return nil, err
	} else {
		return jsonify.Jsonify(incidentRows), err
	}
}

func SendDataAsJSON(w http.ResponseWriter, function string, sqlQry string, args ...any) {
	if data, err := getDatabaseRowsAsJSON(pDB, sqlQry, args...); err != nil {
		ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
	} else {
		if _, err := fmt.Fprint(w, data); err != nil {
			log.Println(err)
		}
	}
}
