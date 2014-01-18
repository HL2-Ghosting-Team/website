var ghostingApp = angular.module('ghostingApp', [
	'ngRoute',
	'ghostingControllers',
	'ghostingServices'
]);

ghostingApp.config(['$routeProvider', '$locationProvider',
	function($routeProvider, $locationProvider) {
		"use strict";

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
	gapi.client.load('ghosting', 'v0', callback, "/_ah/api");
	apisToLoad++;
	gapi.client.load('oauth2', 'v2', callback);

	if(apisToLoad <= 0) {
		callback();
	}
};