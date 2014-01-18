"use strict";

var ghostingApp = angular.module('ghostingApp', [
	'ngRoute',
	'ghostingControllers',
	'ghostingServices'
]);

ghostingApp.config(['$routeProvider', '$locationProvider',
	function($routeProvider, $locationProvider) {
		$locationProvider.html5Mode(true);

		$routeProvider.
			when('/', {
				'templateUrl': '/static/partials/index.html',
				'controller': 'IndexCtrl'
			}).
			when('/logout', {
				'templateUrl': '/static/partials/logout.html',
				'controller': 'LogoutCtrl'
			}).
			otherwise({
				'templateUrl': '/static/partials/not_found.html',
				'controller': 'NotFoundCtrl'
			});
	}
]);

// This function initializes the application by loading the Google APIs.
// TODO: Attempt to improve load performance.
window.initialize = function() {
	var apisToLoad = 0;
	var callback = function() {
		if (--apisToLoad <= 0) {
			//bootstrap manually angularjs after our api are loaded
			angular.bootstrap(document, ["ghostingApp"]);
		}
	}
	apisToLoad++;
	if(location.host.toLowerCase().indexOf("localhost") == 0) {
		gapi.client.load('ghosting', 'v0', callback, "/_ah/api");
		console.log("Ghosting: Using local API.");
	} else {
		gapi.client.load('ghosting', 'v0', callback, "https://ghosting-website.appspot.com/_ah/api");
		console.log("Ghosting: Using production API.");
	}
	apisToLoad++;
	gapi.client.load('oauth2', 'v2', callback);

	if(apisToLoad <= 0) {
		callback();
	}
};