"use strict";

var ghostingControllers = angular.module('ghostingControllers', ['ghostingServices']);

ghostingControllers.controller('HeaderCtrl', ['$scope', 'Auth', '$location', function($scope, Auth, $location) {
	Auth.initialize($scope);

	$scope.isActive = function (viewLocation) { 
        return viewLocation === $location.path();
    };
}]);

ghostingControllers.controller('LogoutCtrl', ['$scope', '$location', function($scope, $location) {
	var takeAction = function() {
		if($scope.auth.user.signed_in) {
			Auth.unauthenticate();
		} else {
			$location.path('/');
		}
	};

	var start = function() {
		takeAction();
		$scope.$watch('auth.user.signed_in', function() {
			takeAction();
		});
	};
	if($scope.auth.loaded) {
		start();
	} else {
		$scope.$watch('auth.loaded', function() {
			if($scope.auth.loaded) start();
		});
	}
}]);