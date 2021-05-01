package apimodel

// SubmitMeasurementRequest is the SubmitMeasurement request.
type SubmitMeasurementRequest struct {
	ReportID string      `path:"report_id"`
	Format   string      `json:"format"`
	Content  interface{} `json:"content"`
}

// SubmitMeasurementResponse is the SubmitMeasurement response.
type SubmitMeasurementResponse struct {
	MeasurementUID string `json:"measurement_uid"`
}
