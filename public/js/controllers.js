var ghostingControllers = angular.module('ghostingControllers', ['ghostingServices']);

ghostingControllers.controller('IndexCtrl', ['$scope', 'Auth', function($scope, Auth) {
	
}]);

ghostingControllers.controller('NotFoundCtrl', ['$scope', 'Auth', function($scope, Auth) {
	
}]);

ghostingControllers.controller('LogoutCtrl', ['$scope', 'Auth', '$location', function($scope, Auth, $location) {
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