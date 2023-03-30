const dfs = {

  setItem: function (val) {
    // we only store high score so we do not use id
    window.updateHighScore(val);
  },

  getItem: async function () {
    const response = await fetch(`https://fairos.staging.fairdatasociety.org/public-kv?key=bestScore&tableName=2048&sharingRef=0256016edfb06dd283c7931fc0ec218ed30581433dc2a77dc3efe3afc495957f`)
      if (response.ok) {
          const data = await response.json();
          return parseInt(atob(data.values));
      }
      return 0;
  },
};

function LocalStorageManager() {
  this.storage = dfs;
}

// Best score getters/setters
LocalStorageManager.prototype.getBestScore = async function () {
    const bestScore = await this.storage.getItem();
  return bestScore || 0;
};

LocalStorageManager.prototype.setBestScore = function (score) {
  this.storage.setItem(score);
};