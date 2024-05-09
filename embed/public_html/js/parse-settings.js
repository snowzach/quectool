function parseCurrentSettings(data) {
    console.log(data);

    // Remove QUIMSLOT and only take 1 or 2
    this.sim = findRegexpValue(data.response, /^\+QUIMSLOT: (\d+)/);
    this.apn = findRegexpValue(data.response, /^\+CGCONTRDP: [\d,]+\"([^"]+)\"/);
    this.cellLock4GStatus = findRegexpValue(data.response, /^\+QNWLOCK: \"common\/4g\",(\d+)/);
    this.cellLock5GStatus = findRegexpValue(data.response, /^\+QNWLOCK: \"common\/5g\",(\d+)/);
    this.prefNetwork = findRegexpValue(data.response, /^\+QNWPREFCFG: \"mode_pref\",(.+)/);
    this.nrModeControlStatus = findRegexpValue(data.response, /^\+QNWPREFCFG: \"nr5g_disable_mode\",(\d+)/);


// 6: '+QCAINFO: "PCC",124350,3,"NR5G BAND 71",869'
// 7: '+QCAINFO: "SCC",520110,12,"NR5G BAND 41",1,290,0,-,-'
// 8: '+QCAINFO: "SCC",502110,6,"NR5G BAND 41",1,290,0,-,-'

    let bands = [];

    const bandRegex = /^\+QCAINFO: (.*)/;
    for (i in data.response) {
      const matches = bandRegex.exec(data.response[i]);
      if(matches) {
        bands.push(matches[1].split(",")[3].replace(/\"/g, " "));
      }
    }

    if (this.cellLock4GStatus == 1 && this.cellLock5GStatus == 1) {
      this.cellLockStatus = "Locked to 4G and 5G";
    } else if (this.cellLock4GStatus == 1) {
      this.cellLockStatus = "Locked to 4G";
    }
    else if (this.cellLock5GStatus == 1) {
      this.cellLockStatus = "Locked to 5G";
    }
    else {
      this.cellLockStatus = "Not Locked";
    }

    if (this.nrModeControlStatus == 0) {
      this.nrModeControlStatus = "Not Disabled";
    }
    else if (this.nrModeControlStatus == 1) {
      this.nrModeControlStatus = "SA Disabled";
    }
    else {
      this.nrModeControlStatus = "NSA Disabled";
    }

    return {
      sim: sim,
      apn: apn,
      cellLockStatus: cellLockStatus,
      prefNetwork: prefNetwork,
      nrModeControl: nrModeControlStatus,
      bands: bands
    };
  }

  function findRegexpValue(array, regexp, index = 1, notFound = "?") {
    for (i in array) {
      const matches = regexp.exec(array[i]);
      if(matches) return matches[index];
    }
    return notFound;
  }
