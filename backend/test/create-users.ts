const http = require('http');

// utils
function post(url, data, token) {
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

  function delete_(url, token) {
    let header = {}
    if(token) {
        header = {
            'Content-Type': 'application/json',
            Authorization: 'Bearer ' + token
        }
    } else {
        header = {
            'Content-Type': 'application/json',
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
  
      req.end()
    })
  }


function get(url, token) {
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

post("http://localhost:3000/users/create", {
        username: "thomas",
        password: "thomasthomas",
        email: "a@gmail.com"
    }, false)
post("http://localhost:3000/users/create", {
        username: "test1",
        password: "thomasthomas",
        email: "a@gmail.com"
    }, false)
post("http://localhost:3000/users/create", {
        username: "test2",
        password: "thomasthomas",
        email: "a@gmail.com"
    }, false)
post("http://localhost:3000/users/create", {
        username: "test3",
        password: "thomasthomas",
        email: "a@gmail.com"
    }, false)
post("http://localhost:3000/users/create", {
        username: "test4",
        password: "thomasthomas",
        email: "a@gmail.com"
    }, false)