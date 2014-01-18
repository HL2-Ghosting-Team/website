"use strict";

var ghostingServices = angular.module('ghostingServices', []);

ghostingServices.factory('GhostingAPI', function() {
	return gapi.client.ghosting;
});

var CLIENT_ID = '434185101696.apps.googleusercontent.com';

var SCOPES = 'https://www.googleapis.com/auth/userinfo.email';

var RESPONSE_TYPE = 'token id_token';

ghostingServices.factory('Auth', [function() {
	var library, authenticateCallback;
	
	authenticateCallback = function($scope) {
		return function() {
			return library.checkAuthenticated($scope);
		}
	};

	library = {
		'initialize': function($scope) {
			$scope.auth = {
				'user': {
					'signed_in': false
				},
				'loaded': false
			}
			library.authenticate(true, $scope);
		},
		'authenticate': function(immediate, $scope) {
			gapi.auth.authorize({
				client_id: CLIENT_ID,
				scope: SCOPES,
				immediate: immediate,
				response_type: RESPONSE_TYPE
			}, authenticateCallback($scope))
		},
		'checkAuthenticated': function($scope) {
			gapi.client.ghosting.users.get({'user': 'current'}).execute(function(resp) {
				if (!resp.code) {
					var token = gapi.auth.getToken();
					token.access_token = token.id_token;
					gapi.auth.setToken(token);

					$scope.auth.user = {
						'signed_in': true,

						'id': resp.id,
						'nickname': resp.nickname,
						'avatar': resp.avatar,
						'admin': resp.admin
					};
					$scope.auth.loaded = true;
					$scope.$apply();
				} else {
					$scope.auth.user = {
						'signed_in': false
					};
					$scope.auth.loaded = false;
					$scope.$apply();
				}
			});
		}
	};

	return library;
}]);