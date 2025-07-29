const http = require('http');
const io = require("socket.io-client");

// utils
function my_post(url, data, token) {
    const dataString = JSON.stringify(data)
    let header = {}
    if(token) {
        header = {
            'Content-Type': 'application/json',
            'Content-Length': dataString.length,
            Authorization: 'Bearer ' + token
        }
    } else {
        header = {
            'Content-Type': 'application/json',
            'Content-Length': dataString.length,
        }
    }

    const options = {
      method: 'POST',
      headers: header,
      timeout: 1000, // in ms
    }
  
    return new Promise((resolve, reject) => {
      const req = http.request(url, options, (res) => {
        if (res.statusCode < 200 || res.statusCode > 299) {
          console.log(res.message)
          return reject(new Error(`HTTP status code ${res.statusCode}: ${res.statusMessage}`))
        }
  
        let body = []
        res.on('data', (chunk) => body.push(chunk))
        res.on('end', () => {
          const resString = Buffer.concat(body).toString()
          resolve(resString)
        })
      })
  
      req.on('error', (err) => {
        reject(err)
      })
  
      req.on('timeout', () => {
        req.destroy()
        reject(new Error('Request time out'))
      })
  
      req.write(dataString)
      req.end()
    })
  }

  function my_delete(url, data, token) {
    const dataString = JSON.stringify(data)
    let header = {}
    if(token) {
        header = {
            'Content-Type': 'application/json',
            'Content-Length': dataString.length,
            Authorization: 'Bearer ' + token
        }
    } else {
        header = {
            'Content-Type': 'application/json',
            'Content-Length': dataString.length,
        }
    }

    const options = {
      method: 'DELETE',
      headers: header,
      timeout: 1000, // in ms
    }
  
    return new Promise((resolve, reject) => {
      const req = http.request(url, options, (res) => {
        if (res.statusCode < 200 || res.statusCode > 299) {
          return reject(new Error(`HTTP status code ${res.statusCode}`))
        }
  
        let body = []
        res.on('data', (chunk) => body.push(chunk))
        res.on('end', () => {
          const resString = Buffer.concat(body).toString()
          resolve(resString)
        })
      })
  
      req.on('error', (err) => {
        reject(err)
      })
  
      req.on('timeout', () => {
        req.destroy()
        reject(new Error('Request time out'))
      })
  
      req.write(dataString)
      req.end()
    })
  }


function my_get(url, token) {
  return new Promise((resolve, reject) => {
    const options = {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`
      }
    };

    const req = http.request(url, options, (res) => {
      let data = '';

      res.on('data', (chunk) => {
        data += chunk;
      });

      res.on('end', () => {
        resolve(data);
      });
    });

    req.on('error', (error) => {
      reject(error);
    });

    req.end();
  });
}

const socket = io("ws://localhost:3000", {
  transports: ['polling', 'websocket'], 
  reconnection: true,
}) 

my_post("http://localhost:3000/users/login", {
        username: "thomas",
        password: "thomasthomas",
    }, false).then((res)=>{
      const token = JSON.parse(res)["access_token"];

    my_get("http://localhost:3000/blinds/get", token).then((result)=>{
      const r = JSON.parse(result)
      //testSockets(r[0].id)
      testYoutube();
    })
    
  })

function testYoutube() {
  socket.on("youtube", (res) => {
    res.items.forEach(element => {
      console.log(element.snippet)
    });
  })
  socket.emit("youtube", {
    query: "gims - ciel"
  })
}

function testSockets(token) {
  let BLIND_TEST_ID = token
  console.log(token)
  console.log('[Client] Connected to server');

  // Join the blind test room
  socket.emit('join', { id: BLIND_TEST_ID });

    
    // Add a folder
    setTimeout(() => {
      socket.emit('addFolder', {
        id: BLIND_TEST_ID,
        name: 'Rock Classics',
        parentId: undefined,
      });
    }, 500);

    
    // Add a music node inside the created folder (simulate it after folder is added)
    socket.on('folderAdded', (folder) => {
      console.log('[Client] Folder added:', folder);
      socket.emit('addMusic', {
        id: BLIND_TEST_ID,
        name: 'Bohemian Rhapsody',
        parentId: folder.id,
        url: 'https://www.youtube.com/watch?v=yk3prd8GER4',
        videoId: 'yk3prd8GER4',
      });
    });
    
    

    socket.on('musicAdded', (music) => {
      console.log('[Client] Music added:', music);

      // Rename the music
      socket.emit('renameNode', {
        id: BLIND_TEST_ID,
        nodeId: music.id.toString(),
        newName: 'Bohemian Rhapsody - Remastered',
      });
    });

    socket.on('nodeRenamed', (renamed) => {
      console.log('[Client] Node renamed:', renamed);

      // Now remove the node
      socket.emit('removeNode', {
        id: BLIND_TEST_ID,
        nodeId: renamed.id.toString(),
      });
    });

    socket.on('nodeRemoved', (payload) => {
      console.log('[Client] Node removed:', payload);
      socket.disconnect();
    });

    socket.on('error', (err) => {
      console.error('[Client] Error from server:', err);
    });
    
  // Callbacks

  socket.on("tree", (tree) => tree.tree.forEach((element)=>{
    console.log("-------");
    displaytree(element, 0);
    console.log("-------");
  }))

  socket.on('disconnect', () => {
    console.log('[Client] Disconnected');
  });
}

function displaytree(tree, prof) {
  let spaces = ""
  for(let i = 0; i < prof; i++) {
    spaces+=" "
  }
  console.log(spaces+tree.name)
  if(tree.childrens) {
    tree.childrens.forEach(element => {
      displaytree(element, prof+1)
    });
  }
}