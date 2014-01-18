"use strict";

var ghostingControllers = angular.module('ghostingControllers', ['ghostingServices']);

ghostingControllers.controller('HeaderCtrl', ['$scope', 'Auth', '$location', function($scope, Auth, $location) {
	Auth.initialize($scope);

	$scope.signIn = function() {
		Auth.authenticate(false, $scope);
	};

	$scope.isActive = function(viewLocation) { 
        return viewLocation === $location.path();
    };
}]);
