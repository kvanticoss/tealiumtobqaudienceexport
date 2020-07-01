package models

import "time"

type AudienceTable struct {
	Updated       time.Time `bigquery:"updated"`
	FullVisitorID string    `bigquery:"fullVisitorId"`
	Audiences     []string  `bigquery:"tealiumAudiences" json:"audiences"`
	Badges        []string  `bigquery:"tealiumBadges" json:"badges"`
	Properties    struct {
		AnalyticsFullVisitor    string `bigquery:"analyticsFullVisitor" json:"ID - Last Analytics Id"`
		AllAnalyticsFullVisitor string `bigquery:"allAnalyticsFullVisitor" json:"ID - All GA Client Ids"`
	} `bigquery:"Ids" json:"properties"`

	TealiumAccount struct {
		name    string `bigquery:"name" json:"account"`
		profile string `bigquery:"profile" json:"profile"`
	} `bigquery:"tealiumAccount" json:"_trace_message_"`
}
