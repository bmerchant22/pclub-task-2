# PClub recruitment Task-2
This is a website created with golang, using gin-gorm for routing purposes and mongoDB for storing github tokens. Also, JWT tokens are used for authentication purposes. A good UI interface could not be implemented, so you need to hit the routes manually on a browser. It is suggested to use Mozilla Firefox or postman for prettier JSON response. 

## Getting Started

These instructions will guide you on how to set up and run the project on your local machine.

### Prerequisites

- [Go](https://golang.org/dl/) installed on your machine

### Installation

1. Clone the repository

   ```bash
   git clone https://github.com/bmerchant22/pclub-task-2.git
   
2. Change to the project directory
   
   ```bash
   cd pclub-task-2
   
3. Install dependencies

   ```bash
   go get ./...

4. Run the main file

   ```bash
   go run cmd/web/main.go
   
## Routes

### Unprotected routes
Routes that can be accessed by a user even if he is not authenticated and logged in

1. **"/"**: This is the home route
2. **"/users/:username"**: On this route, you can view general details about any github user i.e. username, followers, following, public repos etc. You can also view the most popular repo of the user.
3. **"/repos/:username/:repo"**: On this route, you can view any public repo of any user, containing name, desc, createdAt, language and such details.

### Authorization routes
Routes where you need to authorize yourself and github
1. **"/login":** When you hit the login route, you will be redirected to login on github, you need to enter your username and password. Then, after you sign in, you will be taken to a page where github asks you for authorizing my oAuth app.
2. **"/callback":** Once you authorize my oAuth app, you will be redirected to callback route, and this will store your access tokens in the mongodb.

### Protected Routes
These are the route which you can access only if you are verified by github and you have authorized my oAuth app on github. Once you are authorized, the callback will redirect you to the /user route itself
1. **"/user":** On this route, details of the logged in user is shown. This includes the normal user details plus the recent activity and most used language of the user.
2. **"/repos":** On this route, all public and private repos are listed with their names and links to their individual details.
3. **"/private-repos":** On this route, all private repos of the user are listed
4. **"/user-repos/:repo":** On this route, individual repos of the logged-in user can be seen. In addition, list of collaborators and contributors with their individual detail routes can also be seen. Also, the commits can be viewed.

If any error occurs while cloning and running the code, you can contact me at: \
Discord: victorX#7731 \
Mail: bmerchant22@iitk.ac.in
