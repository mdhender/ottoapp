#!/bin/bash
############################################################################
# this script runs ottomap for a clan
#
############################################################################
# 2024/10/04 mdhender update to run as systemd service
# 2024/10/03 mdhender fake a makefile for processing the reports
############################################################################
#
DATA_ROOT=/var/www/ottomap.mdhenderson.com/userdata/
CLAN_ID=

############################################################################
# process command line
#
for arg in "$@"; do
  case "${arg}" in
  --help)
    echo "usage: $0 _clan_id_"
    exit 0;;
  *)
    [ -n "${CLAN_ID}" ] && {
      echo "error: unknown option '${arg}'"
      exit 2
    }
    CLAN_ID="${arg}";;
  esac
done

############################################################################
# verify that we have a clan id and that it is a 4 digit number
#
echo " info: CLAN_ID     == '${CLAN_ID}'"
[ -z "${CLAN_ID}" ] && {
  echo "error: please provide the clan_id to process"
  exit 2
}

############################################################################
# This test derived from https://stackoverflow.com/questions/806906/how-do-i-test-if-a-variable-is-a-number-in-bash
#
re='^0[0-9][0-9][0-9]$'
if ! [[ "${CLAN_ID}" =~ $re ]]; then
  echo "error: clan_id must be a four digit integer"
  exit 2
fi

############################################################################
# set variables for the clan
#
CLAN_ROOT="${DATA_ROOT}${CLAN_ID}/"
echo " info: CLAN_ROOT   == '${CLAN_ROOT}'"
[ -d "${CLAN_ROOT}" ] || {
  echo "error: clan root is not a directory"
  exit 2
}
CLAN_INPUT="${CLAN_ROOT}data/input/"
echo " info: CLAN_INPUT  == '${CLAN_INPUT}'"
[ -d "${CLAN_INPUT}" ] || {
  echo "error: clan input is not a directory"
  exit 2
}
CLAN_LOGS="${CLAN_ROOT}data/logs/"
echo " info: CLAN_LOGS   == '${CLAN_LOGS}'"
[ -d "${CLAN_LOGS}" ] || {
  echo "error: clan logs is not a directory"
  exit 2
}
CLAN_OUTPUT="${CLAN_ROOT}data/output/"
echo " info: CLAN_OUTPUT == '${CLAN_OUTPUT}'"
[ -d "${CLAN_OUTPUT}" ] || {
  echo "error: clan output is not a directory"
  exit 2
}

############################################################################
# set def to the clan input and see what we need to do. note that this can
# introduce a race condition if the user uploads a file after we've found
# the "current" report but before we run the render. oh well.
#
# also, we assume that the for loop sorts the report files. bash should.
#
cd "${CLAN_INPUT}" || {
  echo "error: clan input exists but we can't set def to it"
  exit 2
}
CURR_TURN=
reports=
turns=
for file in 0???-??.0???.report.txt dummy; do
  [ -f "${file}" ] || continue
  CURR_TURN="${file%.0???.report.txt}"
  reports="${reports} ${file}"
  turns="${turns} ${CURR_TURN}"
done
[ -z "${CURR_TURN}" ] && {
  echo " info: did not find a turn to process"
  exit 0
}
echo " info: reports ${reports}"
echo " info: turns   ${turns}"
echo " info: CURR_TURN   == '${CURR_TURN}'"

#CLAN_LOG="${CLAN_LOGS}${turn}.${CLAN_ID}.log"
#echo " info: CLAN_LOG    == '${CLAN_LOG}'"
#[ -f "${CLAN_LOG}" ] && {
#  echo " info: removing log file from prior run"
#  rm -f "${CLAN_LOG}"
#}

#CLAN_ERROR="${CLAN_LOGS}${turn}.${CLAN_ID}.err"
#echo " info: CLAN_ERROR  == '${CLAN_ERROR}'"
#[ -f "${CLAN_ERROR}" ] && {
#  echo " info: removing error tag from prior run"
#  rm -f "${CLAN_ERROR}"
#}

############################################################################
# set def to the root of the clan data and run the render, saving the log
# or the error log.
#
cd "${CLAN_ROOT}" || {
  echo "error: clan root exists but we can't set def to it"
  exit 2
}

for turn in ${turns} dummy; do
  [ "${turn}" == dummy ] && break
  echo
  echo
  echo " info: turn       == '${turn}'..."

  reportFile="${CLAN_INPUT}${turn}.${CLAN_ID}.report.txt"
  mapFile="${CLAN_OUTPUT}${turn}.${CLAN_ID}.wxx"
  logFile="${CLAN_LOGS}${turn}.${CLAN_ID}.log"
  errorFile="${CLAN_LOGS}${turn}.${CLAN_ID}.err"
  tmpFile="${CLAN_LOGS}${turn}.${CLAN_ID}.tmp"

  echo " info: reportFile == '${reportFile}"
  echo " info: mapFile    == '${mapFile}"
  echo " info: logFile    == '${logFile}"
  echo " info: errorFile  == '${errorFile}"
  echo " info: tmpFile    == '${tmpFile}"

  # set renderMap to NO initially
  renderMap=NO
  # 1. If we don't have a log file or error file, set renderMap to YES
  if [ ! -f "$logFile" ] && [ ! -f "$errorFile" ]; then
    renderMap=YES
  # 2. Else if we have an error file and the report file is newer than the error file, set renderMap to YES
  elif [ -f "$errorFile" ] && [ "$reportFile" -nt "$errorFile" ]; then
    renderMap=YES
  # 3. Else if we have a log file and the report file is newer than the log file, set renderMap to YES
  elif [ -f "$logFile" ] && [ "$reportFile" -nt "$logFile" ]; then
    renderMap=YES
  fi

  if [ "${renderMap}" == YES ]; then
    echo " info: running the map render"
    rm -f "${errorFile}" "${logFile}" "${tmpFile}"
    /var/www/ottomap.mdhenderson.com/bin/ottomap render --log-file "${tmpFile}" --clan-id "${CLAN_ID}" --max-turn "${turn}" --show-grid-coords --shift-map --save-with-turn-id --auto-eol
    if [ $? != 0 ]; then
      echo "error: render failed, leaving error file"
      cp -p "${tmpFile}" "${errorFile}"
      exit 2
    fi
    echo " info: render succeeded, leaving log file"
    cp -p "${tmpFile}" "${logFile}"
  fi
done

############################################################################
#
echo " info: ottomap completed successfully"
exit 0
