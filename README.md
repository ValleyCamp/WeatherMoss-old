# WeatherMoss

The official weather API of [ValleyCamp](http://valleycamp.org). Visit the [Dashboard](http://actual.valleycamp.org/weathermoss/gui/freeboard/#source=dashboard.json) to see it in action.

On-Site we have a Davis Vantage Pro 2 weather station and a [MeteoBridge](http://meteobridge.com). In addition to uploading data to various weather services the Meteobridge device will log data into a MySQL database, which this API will call against. We may also try to have the API pull live data directly from the meteobridge.

The application is a Go binary compiled for freebsd and deployed on-site. It provides JSON data via the REST API and WebSocket connections. There may be connection issues due to the nature of the "High Speed" internet connection on-site, so applications calling against the API should expect potential network dropouts or high latency.


## MySQL Database.
A SQL database which the meteobridge will dump data into.
Database inspired by http://www.stevejenkins.com/blog/2015/02/storing-weather-station-data-mysql-meteobridge/

The tables we're using are defined as follows:

```sql
CREATE TABLE IF NOT EXISTS `housestation_10min_all` (
  `ID` int(11) NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `DateTime` datetime NOT NULL COMMENT 'Date and Time of Readings',
  `TempOutCur` decimal(4,1) NOT NULL COMMENT 'Current Outdoor Temperature',
  `HumOutCur` int(11) NOT NULL COMMENT 'Current Outdoor Humidity',
  `PressCur` decimal(4,2) NOT NULL COMMENT 'Current Barometric Pressure',
  `DewCur` decimal(4,1) NOT NULL COMMENT 'Current Dew Point',
  `HeatIdxCur` decimal(4,1) NOT NULL COMMENT 'Current Heat Index',
  `WindChillCur` decimal(4,1) NOT NULL COMMENT 'Current Wind Chill',
  `TempInCur` decimal(4,1) NOT NULL COMMENT 'Current Indoor Temperature',
  `HumInCur` int(11) NOT NULL COMMENT 'Current Indoor Humidity',
  `WindSpeedCur` decimal(4,1) NOT NULL COMMENT 'Current Wind Speed',
  `WindAvgSpeedCur` decimal(4,1) NOT NULL COMMENT 'Current Average Wind Speed',
  `WindDirCur` int(11) NOT NULL COMMENT 'Current Wind Direction (Degrees)',
  `WindDirCurEng` varchar(3) NOT NULL COMMENT 'Current Wind Direction (English)',
  `WindGust10` decimal(4,1) NOT NULL COMMENT 'Max Wind Gust for Past 10 Mins',
  `WindDirAvg10` int(11) NOT NULL COMMENT 'Average Wind Direction (Degrees) for Past 10 Mins',
  `WindDirAvg10Eng` varchar(3) NOT NULL COMMENT 'Average Wind Direction (English) for Past 10 Mins',
  `UVAvg10` decimal(6,2) NOT NULL COMMENT 'Average UV Level for past 10 Mins',
  `UVMax10` decimal(6,2) NOT NULL COMMENT 'Max UV Level for past 10 Mins',
  `SolarRadAvg10` decimal(6,2) NOT NULL COMMENT 'Average Solar Radiation for past 10 Mins',
  `SolarRadMax10` decimal(6,2) NOT NULL COMMENT 'Max Solar Radiation for past 10 Mins',
  `RainRateCur` decimal(5,2) NOT NULL COMMENT 'Current Rain Rate',
  `RainDay` decimal(4,2) NOT NULL COMMENT 'Total Rain for Today',
  `RainYest` decimal(4,2) NOT NULL COMMENT 'Total Rain for Yesterday',
  `RainMonth` decimal(5,2) NOT NULL COMMENT 'Total Rain this Month',
  `RainYear` decimal(5,2) NOT NULL COMMENT 'Total Rain this Year'
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=latin1;
-- Execute this SQL in meteobridge as an every-10-minute event:
-- INSERT INTO `housestation_10min_all` (`DateTime`, `TempOutCur`, `HumOutCur`, `PressCur`, `DewCur`, `HeatIdxCur`, `WindChillCur`, `TempInCur`, `HumInCur`, `WindSpeedCur`, `WindAvgSpeedCur`, `WindDirCur`, `WindDirCurEng`, `WindGust10`, `WindDirAvg10`, `WindDirAvg10Eng`, `UVAvg10`, `UVMax10`, `SolarRadAvg10`, `SolarRadMax10`, `RainRateCur`, `RainDay`, `RainYest`, `RainMonth`, `RainYear`) VALUES ('[YYYY]-[MM]-[DD] [hh]:[mm]:[ss]', '[th0temp-act=F]', '[th0hum-act]', '[thb0seapress-act=inHg.2]', '[th0dew-act=F]', '[th0heatindex-act=F]', '[wind0chill-act=F]', '[thb0temp-act=F]', '[thb0hum-act]', '[wind0wind-act=mph]', '[wind0avgwind-act=mph]', '[wind0dir-act]', '[wind0dir-act=endir]', '[wind0wind-max10=mph]', '[wind0dir-avg10]', '[wind0dir-avg10=endir]', '[uv0index-avg10]', '[uv0index-max10]', '[sol0rad-avg10]', '[sol0rad-max10]', '[rain0rate-act=in.2]', '[rain0total-daysum=in.2]', '[rain0total-ydaysum=in.2]', '[rain0total-monthsum=in.2]', '[rain0total-yearsum=in.2]')

CREATE TABLE IF NOT EXISTS `housestation_15sec_wind` (
  `ID` int(11) NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `DateTime` datetime NOT NULL COMMENT 'Date and Time of Result',
  `WindDirCur` int(11) NOT NULL COMMENT 'Wind Direction (Degrees) at this instant',
  `WindDirCurEng` varchar(3) NOT NULL COMMENT 'Wind Direction (English) at this instant',
  `WindSpeedCur` decimal(4,1) NOT NULL COMMENT 'Wind Speed at this instant'
);
-- Execute this SQL in meteobridge as an every-15-seconds event:
-- INSERT INTO `housestation_15sec_wind` (`DateTime`, `WindDirCur`, `WindDirCurEng`, `WindSpeedCur`) VALUES ('[YYYY]-[MM]-[DD] [hh]:[mm]:[ss]', '[wind0dir-act]', '[wind0dir-act=endir]', '[wind0wind-act=mph]')
```

